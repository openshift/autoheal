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
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/golang/glog"
	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	"github.com/jhernand/openshift-monitoring/pkg/client/openshift"
	"github.com/jhernand/openshift-monitoring/pkg/labels"
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

	// Publish or retire the alert, according to the status provided by the alert manager:
	switch alert.Status {
	case alertmanager.AlertStatusFiring:
		p.publishAlert(object)
	case alertmanager.AlertStatusResolved:
		p.resolveAlert(object)
	default:
		glog.Warningf(
			"Unknnown status '%s' reported by alert manager, will ignore it",
			alert.Status,
		)
	}
}

func (p *Publisher) publishAlert(alert *monitoring.Alert) {
	// Add the firing condition:
	alert.AddCondition(monitoring.AlertFiring)

	// Add the hash to the name:
	hash := alert.ObjectMeta.Labels[labels.Hash]
	name := fmt.Sprintf("%s-%s", alert.ObjectMeta.Name, hash)

	// Repeatedly try to save the alert till we find a suffix that makes the name unique:
	suffix := 0
	for {
		alert.ObjectMeta.Name = fmt.Sprintf("%s-%d", name, suffix)
		err := p.tryPublishAlert(alert)
		if err == nil {
			break
		} else if errors.IsAlreadyExists(err) {
			suffix++
			continue
		} else {
			glog.Infof(
				"Can't publish alert '%s': %s",
				name,
				err.Error(),
			)
			return
		}
	}
}

func (p *Publisher) tryPublishAlert(alert *monitoring.Alert) error {
	// Get the resource that manages the set of alerts:
	resource := p.client.Monitoring().Alerts(alert.ObjectMeta.Namespace)

	// Try to find an existing alert that matches the new one, and move it to firing:
	match, err := p.findMatch(alert)
	if err != nil {
		return err
	}
	if match != nil {
		firing := match.HasCondition(monitoring.AlertFiring)
		resolved := match.HasCondition(monitoring.AlertResolved)
		if !firing || resolved {
			match.DeleteCondition(monitoring.AlertResolved)
			match.AddCondition(monitoring.AlertFiring)
			_, err := resource.Update(match)
			if err != nil {
				return err
			}
		}
		glog.Infof(
			"Alert was already published as '%s'",
			match.ObjectMeta.Name,
		)
		return nil
	}

	// Save the new alert:
	_, err = resource.Create(alert)
	if err != nil {
		return err
	} else {
		glog.Infof(
			"Alert '%s' has been published",
			alert.ObjectMeta.Name,
		)
		return nil
	}

	return nil
}

func (p *Publisher) resolveAlert(alert *monitoring.Alert) {
	// Get the resource that manages the set of alerts:
	resource := p.client.Monitoring().Alerts(alert.ObjectMeta.Namespace)

	// Try to find an existing alert that matches the resolved one:
	match, err := p.findMatch(alert)
	if err != nil {
		glog.Errorf(
			"Can't find alerts matching '%s': %s",
			alert.ObjectMeta.Name,
			err.Error(),
		)
		return
	}
	if match == nil {
		glog.Infof(
			"Alert '%s' doesn't exist",
			alert.ObjectMeta.Name,
		)
		return
	}

	// Try to move the alert to resolved:
	firing := match.HasCondition(monitoring.AlertFiring)
	resolved := match.HasCondition(monitoring.AlertResolved)
	if firing || !resolved {
		match.DeleteCondition(monitoring.AlertFiring)
		match.AddCondition(monitoring.AlertResolved)
		_, err := resource.Update(match)
		if err != nil {
			glog.Infof(
				"Can't update conditions of alert '%s': %s",
				match.ObjectMeta.Name,
				err.Error(),
			)
			return
		}
		glog.Infof(
			"Alert '%s' has been resolved",
			match.ObjectMeta.Name,
		)
	} else {
		glog.Infof(
			"Alert '%s' was already resolved",
			match.ObjectMeta.Name,
		)
	}
}

func (p *Publisher) findMatch(alert *monitoring.Alert) (match *monitoring.Alert, err error) {
	// Load all the alerts that have the same hash than the one that we are looking for:
	resource := p.client.Monitoring().Alerts(alert.ObjectMeta.Namespace)
	hash := alert.ObjectMeta.Labels[labels.Hash]
	options := meta.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labels.Hash, hash),
	}
	list, err := resource.List(options)
	if err != nil {
		return
	}

	// Check if any of the existing alerts has the same identitity than the new one. If there is
	// such alert then return it:
	for _, item := range list.Items {
		same := reflect.DeepEqual(item.Status.Labels, alert.Status.Labels)
		if same {
			match = &item
			return
		}
	}

	return
}

func (p *Publisher) indent(data []byte) []byte {
	buffer := new(bytes.Buffer)
	err := json.Indent(buffer, data, "", "  ")
	if err != nil {
		return data
	}
	return buffer.Bytes()
}
