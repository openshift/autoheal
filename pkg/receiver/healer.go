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

package receiver

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
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"

	"github.com/openshift/autoheal/pkg/config"
	"github.com/openshift/autoheal/pkg/memory"
	"github.com/openshift/autoheal/pkg/receiver/alertmanager"
)

// HealerBuilder is used to create new healers.
//
type HealerBuilder struct {
	// Configuration files.
	configFiles []string

	// Kubernetes client.
	k8sClient kubernetes.Interface
}

// Healer contains the information needed to receive notifications about changes in the
// Prometheus configuration and to start or reload it when there are changes.
//
type Healer struct {
	// The configuration.
	config *config.Config

	// Kubernetes client.
	k8sClient kubernetes.Interface

	// The current set of healing rules.
	rulesCache *syncmap.Map

	// We use two queues, one to process updates to the rules and another to process incoming
	// notifications from the alert manager:
	rulesQueue  workqueue.RateLimitingInterface
	alertsQueue workqueue.RateLimitingInterface

	// Executed actions will be stored here in order to prevent repeated execution.
	actionMemory *memory.ShortTermMemory

	// The AWX active jobs
	activeJobs *syncmap.Map
}

// NewHealerBuilder creates a new builder for healers.
//
func NewHealerBuilder() *HealerBuilder {
	b := new(HealerBuilder)
	return b
}

// ConfigFile adds one configuration file.
//
func (b *HealerBuilder) ConfigFile(path string) *HealerBuilder {
	b.configFiles = append(b.configFiles, path)
	return b
}

// ConfigFiles adds one or more configuration files or directories. They will be loaded in the order
// given. For directories all the contained files will be loaded, in alphabetical order.
//
func (b *HealerBuilder) ConfigFiles(paths []string) *HealerBuilder {
	if len(paths) > 0 {
		for _, path := range paths {
			b.ConfigFile(path)
		}
	}
	return b
}

// KubernetesClient sets the Kubernetes client that will be used by the healer.
//
func (b *HealerBuilder) KubernetesClient(client kubernetes.Interface) *HealerBuilder {
	b.k8sClient = client
	return b
}

// Build creates the healer using the configuration stored in the builder.
//
func (b *HealerBuilder) Build() (h *Healer, err error) {
	// Load the configuration files:
	if len(b.configFiles) == 0 {
		err = fmt.Errorf("No configuration file has been provided")
		return
	}
	config, err := config.NewLoader().
		Client(b.k8sClient).
		Files(b.configFiles).
		Load()
	if err != nil {
		return
	}

	// Send to the log a summary of the configuration:
	glog.Infof("AWX user is '%s'", config.AWX().User())
	glog.Infof("AWX project is '%s'", config.AWX().Project())

	// Create the actions memory:
	actionMemory, err := memory.NewShortTermMemoryBuilder().
		Duration(config.Throttling().Interval()).
		Build()
	if err != nil {
		return
	}

	// Allocate the healer:
	h = new(Healer)
	h.k8sClient = b.k8sClient
	h.config = config
	h.actionMemory = actionMemory

	// Initialize the map of rules:
	h.rulesCache = new(syncmap.Map)

	// Create the queues:
	h.rulesQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rules")
	h.alertsQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "alerts")

	// initialize active jobs map:
	h.activeJobs = new(syncmap.Map)
	return
}

// Run waits for the informers caches to sync, and then starts the workers and the web server.
//
func (h *Healer) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer h.rulesQueue.ShutDown()
	defer h.alertsQueue.ShutDown()

	// Start the workers:
	go wait.Until(h.runRulesWorker, time.Second, stopCh)
	go wait.Until(h.runAlertsWorker, time.Second, stopCh)
	go wait.Until(h.runActiveJobsWorker, h.config.AWX().JobStatusCheckInterval(), stopCh)
	glog.Info("Workers started")

	// For each rule inside the configuration create a change and add it to the queue:
	rules := h.config.Rules()
	if len(rules) > 0 {
		for _, rule := range rules {
			change := &RuleChange{
				Type: watch.Added,
				Rule: rule,
			}
			h.rulesQueue.Add(change)
		}
		glog.Infof("Loaded %d healing rules from the configuration", len(rules))
	} else {
		glog.Warningf("There are no healing rules in the configuration")
	}

	// Start the web server:
	http.Handle("/metrics", h.metricsHandler())
	http.HandleFunc("/alerts", h.handleRequest)

	server := &http.Server{Addr: ":9099"}
	go server.ListenAndServe()
	glog.Info("Web server started")

	// Wait till we are requested to stop:
	<-stopCh

	// Shutdown the web server:
	err := server.Shutdown(context.TODO())
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
	if glog.V(2) {
		glog.Infof("Request body:\n%s", h.indent(body))
	}

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
