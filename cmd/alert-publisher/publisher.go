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
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	"github.com/jhernand/openshift-monitoring/pkg/client/openshift"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
)

// PublisherBuilders is used to create new publishers.
//
type PublisherBuilder struct {
	// Client.
	client openshift.Interface
}

// Publisher contains the information needed to receive notifications from the Prometheus alert
// manager and to publish them as Kubernetes alert objects.
//
type Publisher struct {
	// Client.
	client openshift.Interface
}

// NewPublisherBuilder creates a new builder for publishers.
//
func NewPublisherBuilder() *PublisherBuilder {
	b := new(PublisherBuilder)
	return b
}

// Client sets the OpenShift client that will be used by the publisher.
//
func (b *PublisherBuilder) Client(client openshift.Interface) *PublisherBuilder {
	b.client = client
	return b
}

// Build creates the publisher using the configuration stored in the builder.
//
func (b *PublisherBuilder) Build() (p *Publisher, err error) {
	// Allocate the publisher:
	p = new(Publisher)

	// Save the reference to the OpenShift client:
	p.client = b.client

	return
}

// Run runs the publisher.
//
func (p *Publisher) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// Register the handlers and start listening:
	http.HandleFunc("/", p.handleRequest)
	http.ListenAndServe(":9099", nil)

	// Wait till we are requested to stop:
	<-stopCh

	return nil
}

func (p *Publisher) handleRequest(response http.ResponseWriter, request *http.Request) {
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
	glog.Infof("Request body:\n%s", p.indent(body))

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
	p.handleMessage(message)
}

func (p *Publisher) handleMessage(message *alertmanager.Message) {
	for _, alert := range message.Alerts {
		p.handleAlert(alert)
	}
}

func (p *Publisher) handleAlert(alert *alertmanager.Alert) {
	// Transform the alert data into the corresponding Kubernetes object:
	object, err := mapAlert(alert)
	if err != nil {
		glog.Errorf("Can't transform alert: %s", err.Error())
		return
	}

	// Publish the alert:
	p.publishAlert(object)
}

func (p *Publisher) publishAlert(alert *monitoring.Alert) {
	// Get the resource that manages the collection of alerts for the namespace:
	namespace := alert.ObjectMeta.Namespace
	if namespace == "" {
		namespace = "default"
	}
	resource := p.client.Monitoring().Alerts(namespace)

	// Try to create or update the alert:
	_, err := resource.Create(alert)
	if errors.IsAlreadyExists(err) {
		loaded, err := resource.Get(alert.ObjectMeta.Name, meta.GetOptions{})
		if err != nil {
			glog.Errorf("Can't retrieve alert '%s': %s", alert.ObjectMeta.Name, err.Error())
		}
		alert.ObjectMeta.ResourceVersion = loaded.ObjectMeta.ResourceVersion
		_, err = resource.Update(alert)
		if err != nil {
			glog.Errorf("Can't update alert '%s': $s", alert.ObjectMeta.Name, err.Error())
		} else {
			glog.Infof("Alert '%s' has been updated", alert.ObjectMeta.Name)
		}
	} else if err != nil {
		glog.Errorf("Can't create alert '%s': %s", alert.ObjectMeta.Name, err.Error())
	} else {
		glog.Infof("Alert '%s' has been created", alert.ObjectMeta.Name)
	}
}

func (p *Publisher) indent(data []byte) []byte {
	buffer := new(bytes.Buffer)
	err := json.Indent(buffer, data, "", "  ")
	if err != nil {
		return data
	}
	return buffer.Bytes()
}
