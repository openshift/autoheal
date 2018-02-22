/*
Copyright (c) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/glog"
	"golang.org/x/sync/syncmap"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	genericinf "github.com/jhernand/openshift-monitoring/pkg/client/informers"
	typedinf "github.com/jhernand/openshift-monitoring/pkg/client/informers/monitoring/v1alpha1"
	openshift "github.com/jhernand/openshift-monitoring/pkg/client/openshift"
)

// HealerBuilder is used to create new healers.
//
type HealerBuilder struct {
	// Clients.
	k8sClient kubernetes.Interface
	osClient  openshift.Interface

	// Informer factory.
	informerFactory genericinf.SharedInformerFactory
}

// Healer contains the information needed to receive notifications about changes in the
// Prometheus configuration and to start or reload it when there are changes.
//
type Healer struct {
	// Client.
	k8sClient kubernetes.Interface
	osClient  openshift.Interface

	// Informers.
	ruleInformer typedinf.HealingRuleInformer

	// The current set of healing rules.
	rulesCache *syncmap.Map

	// We use two queues, one to process updates to the rules and another to process incoming
	// notifications from the alert manager:
	rulesQueue  workqueue.RateLimitingInterface
	alertsQueue workqueue.RateLimitingInterface
}

// NewHealerBuilder creates a new builder for healers.
//
func NewHealerBuilder() *HealerBuilder {
	b := new(HealerBuilder)
	return b
}

// KubernetesClient sets the Kubernetes client that will be used by the healer.
//
func (b *HealerBuilder) KubernetesClient(client kubernetes.Interface) *HealerBuilder {
	b.k8sClient = client
	return b
}

// OpenShiftClient sets the OpenShift client that will be used by the healer.
//
func (b *HealerBuilder) OpenShiftClient(client openshift.Interface) *HealerBuilder {
	b.osClient = client
	return b
}

// InformerFactory sets the OpenShift informer factory that will be used by the healer.
//
func (b *HealerBuilder) InformerFactory(factory genericinf.SharedInformerFactory) *HealerBuilder {
	b.informerFactory = factory
	return b
}

// Build creates the healer using the configuration stored in the builder.
//
func (b *HealerBuilder) Build() (h *Healer, err error) {
	// Allocate the healer:
	h = new(Healer)

	// Save the references to the clients:
	h.k8sClient = b.k8sClient
	h.osClient = b.osClient

	// Save the reference to the informer factory:

	// Initialize the map of rules:
	h.rulesCache = new(syncmap.Map)

	// Create the queues:
	h.rulesQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rules")
	h.alertsQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "alerts")

	// Set up an event handler to detect changes in the set of healing rules:
	h.ruleInformer = b.informerFactory.Monitoring().V1alpha1().HealingRules()
	h.ruleInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				switch typed := obj.(type) {
				case *monitoring.HealingRule:
					change := &RuleChange{
						Type: watch.Added,
						Rule: typed,
					}
					h.rulesQueue.AddRateLimited(change)
				}
			},
			UpdateFunc: func(_, obj interface{}) {
				switch typed := obj.(type) {
				case *monitoring.HealingRule:
					change := &RuleChange{
						Type: watch.Modified,
						Rule: typed,
					}
					h.rulesQueue.AddRateLimited(change)
				}
			},
			DeleteFunc: func(obj interface{}) {
				switch typed := obj.(type) {
				case *monitoring.HealingRule:
					change := &RuleChange{
						Type: watch.Deleted,
						Rule: typed,
					}
					h.rulesQueue.AddRateLimited(change)
				}
			},
		},
	)

	return
}

// Run waits for the informers caches to sync, and then starts the workers and the web server.
//
func (h *Healer) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer h.rulesQueue.ShutDown()
	defer h.alertsQueue.ShutDown()

	// Wait for the caches to be synced before starting the workers:
	glog.Info("Waiting for informer caches to sync")
	rulesSynced := h.ruleInformer.Informer().HasSynced
	if ok := cache.WaitForCacheSync(stopCh, rulesSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}
	glog.Info("Informer caches are in sync")

	// Populate the rules cache:
	rules, err := h.ruleInformer.Lister().List(labels.Everything())
	if err != nil {
		return err
	}
	for _, rule := range rules {
		h.rulesCache.Store(rule.ObjectMeta.Name, rule)
		glog.Infof("Loaded rule '%s'", rule.ObjectMeta.Name)
	}

	// Start the workers:
	go wait.Until(h.runRulesWorker, time.Second, stopCh)
	go wait.Until(h.runAlertsWorker, time.Second, stopCh)
	glog.Info("Workers started")

	// Start the web server:
	http.HandleFunc("/", h.handleRequest)
	server := &http.Server{Addr: ":9099"}
	go server.ListenAndServe()
	glog.Info("Web server started")

	// Wait till we are requested to stop:
	<-stopCh

	// Shutdown the web server:
	err = server.Shutdown(context.TODO())
	if err != nil {
		return err
	}

	return nil
}

func (h *Healer) handleRequest(response http.ResponseWriter, request *http.Request) {
	// Read the request body:
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		glog.Warningf("Can't read request body: %s", err)
		http.Error(
			response,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	// Dump the request to the log:
	glog.Infof("Request body:\n%s", h.indent(body))

	// Parse the JSON request body:
	message := new(alertmanager.Message)
	json.Unmarshal(body, message)
	if err != nil {
		glog.Warningf("Can't parse request body: %s", err)
		http.Error(
			response,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	// Handle the parsed message:
	h.handleMessage(message)
}

func (h *Healer) handleMessage(message *alertmanager.Message) {
	for _, alert := range message.Alerts {
		h.alertsQueue.AddRateLimited(alert)
	}
}

func (h *Healer) indent(data []byte) []byte {
	buffer := new(bytes.Buffer)
	err := json.Indent(buffer, data, "", "  ")
	if err != nil {
		return data
	}
	return buffer.Bytes()
}
