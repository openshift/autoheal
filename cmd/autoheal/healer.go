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
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/glog"
	"golang.org/x/sync/syncmap"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"

	"github.com/openshift/autoheal/pkg/alertmanager"
)

// HealerBuilder is used to create new healers.
//
type HealerBuilder struct {
	// Clients.
	k8sClient kubernetes.Interface
}

// Healer contains the information needed to receive notifications about changes in the
// Prometheus configuration and to start or reload it when there are changes.
//
type Healer struct {
	// Client.
	k8sClient kubernetes.Interface

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

// Build creates the healer using the configuration stored in the builder.
//
func (b *HealerBuilder) Build() (h *Healer, err error) {
	// Allocate the healer:
	h = new(Healer)

	// Save the references to the clients:
	h.k8sClient = b.k8sClient

	// Initialize the map of rules:
	h.rulesCache = new(syncmap.Map)

	// Create the queues:
	h.rulesQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rules")
	h.alertsQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "alerts")

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
	glog.Info("Workers started")

	// Start the web server:
	http.HandleFunc("/", h.handleRequest)
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
