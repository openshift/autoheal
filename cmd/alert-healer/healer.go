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
	"fmt"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	awx "github.com/jhernand/openshift-monitoring/pkg/awx"
	informers "github.com/jhernand/openshift-monitoring/pkg/client/informers"
	listers "github.com/jhernand/openshift-monitoring/pkg/client/listers/monitoring/v1alpha1"
	openshift "github.com/jhernand/openshift-monitoring/pkg/client/openshift"
)

// HealerBuilder is used to create new healers.
//
type HealerBuilder struct {
	// Client.
	client openshift.Interface

	// Informer factory.
	informerFactory informers.SharedInformerFactory
}

// Healer contains the information needed to receive notifications about changes in the
// Prometheus configuration and to start or reload it when there are changes.
//
type Healer struct {
	// Clients.
	client openshift.Interface

	// Informer factory.
	informerFactory informers.SharedInformerFactory

	// Listers.
	alertLister       listers.AlertLister
	healingRuleLister listers.HealingRuleLister

	// Sync checkers.
	alertsSynced       cache.InformerSynced
	healingRulesSynced cache.InformerSynced
}

// NewHealerBuilder creates a new builder for healers.
//
func NewHealerBuilder() *HealerBuilder {
	b := new(HealerBuilder)
	return b
}

// Client sets the OpenShift client that will be used by the healer.
//
func (b *HealerBuilder) Client(client openshift.Interface) *HealerBuilder {
	b.client = client
	return b
}

// InformerFactory sets the OpenShift informer factory that will be used by the healer.
//
func (b *HealerBuilder) InformerFactory(factory informers.SharedInformerFactory) *HealerBuilder {
	b.informerFactory = factory
	return b
}

// Build creates the healer using the configuration stored in the builder.
//
func (b *HealerBuilder) Build() (h *Healer, err error) {
	// Allocate the healer:
	h = new(Healer)

	// Save the references to the OpenShift client:
	h.client = b.client

	// Get references to the informer:
	alertInformer := b.informerFactory.Monitoring().V1alpha1().Alerts()
	h.alertLister = alertInformer.Lister()
	h.alertsSynced = alertInformer.Informer().HasSynced
	healingRuleInformer := b.informerFactory.Monitoring().V1alpha1().HealingRules()
	h.healingRuleLister = healingRuleInformer.Lister()
	h.healingRulesSynced = healingRuleInformer.Informer().HasSynced

	// Set up an event handler for when alerts are created, modified or deleted:
	alertInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				switch change := obj.(type) {
				case *monitoring.Alert:
					glog.Infof(
						"Alert '%s' has been added",
						change.ObjectMeta.Name,
					)
					h.handleAlertChange(change)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				switch change := newObj.(type) {
				case *monitoring.Alert:
					glog.Infof(
						"Alert '%s' has been updated",
						change.ObjectMeta.Name,
					)
					h.handleAlertChange(change)
				}
			},
			DeleteFunc: func(obj interface{}) {
				switch change := obj.(type) {
				case *monitoring.Alert:
					glog.Infof(
						"Alert '%s' has been deleted",
						change.ObjectMeta.Name,
					)
					h.handleAlertChange(change)
				}
			},
		},
	)

	return
}

// Run waits for the informers caches to sync, and then starts the healer.
//
func (h *Healer) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// Wait for the caches to be synced before starting the worker:
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, h.alertsSynced, h.healingRulesSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	// Wait till we are requested to stop:
	<-stopCh

	return nil
}

// handleAlertChange checks if the given alert change requires starting a healing process.
//
func (h *Healer) handleAlertChange(alert *monitoring.Alert) {
	// Load the healing rules:
	rules, err := h.healingRuleLister.List(labels.Everything())
	if err != nil {
		glog.Info("Can't load healing rules: %s", err.Error())
	}

	// Find the rules that are activated for the alert:
	activated := make([]*monitoring.HealingRule, 0)
	for _, rule := range rules {
		if h.checkConditions(rule, alert) {
			glog.Infof(
				"Healing rule '%s' matches alert '%s'",
				rule.ObjectMeta.Name,
				alert.ObjectMeta.Name,
			)
			activated = append(activated, rule)
		}
	}
	if len(activated) == 0 {
		glog.Infof("No healing rule matches alert '%s'", alert.ObjectMeta.Name)
		return
	}

	// Execute the actions of the activated rules:
	for _, rule := range activated {
		h.runActions(rule, alert)
	}
}

func (h *Healer) checkConditions(rule *monitoring.HealingRule, alert *monitoring.Alert) bool {
	if rule.Spec.Conditions != nil && len(rule.Spec.Conditions) > 0 {
		for i := 0; i < len(rule.Spec.Conditions); i++ {
			if !h.checkCondition(&rule.Spec.Conditions[i], alert) {
				return false
			}
		}
	}
	return true
}

func (h *Healer) checkCondition(condition *monitoring.HealingCondition, alert *monitoring.Alert) bool {
	return condition.Alert == alert.ObjectMeta.Name
}

func (h *Healer) runActions(rule *monitoring.HealingRule, alert *monitoring.Alert) {
	if rule.Spec.Actions != nil && len(rule.Spec.Actions) > 0 {
		glog.Infof(
			"Running actions of healing rule '%s' for alert '%s'",
			rule.ObjectMeta.Name,
			alert.ObjectMeta.Name,
		)
		for i := 0; i < len(rule.Spec.Actions); i++ {
			h.runAction(&rule.Spec.Actions[i], alert)
		}
	} else {
		glog.Warningf(
			"Healing rule '%s' has no actions, will have no effect on alert '%s'",
			rule.ObjectMeta.Name,
			alert.ObjectMeta.Name,
		)
	}
}

func (h *Healer) runAction(action *monitoring.HealingAction, alert *monitoring.Alert) {
	if action.AWX != nil {
		h.runAWX(action.AWX, alert)
	} else {
		glog.Warningf(
			"There are no action details, action will have no effect on alert '%s'",
			alert.ObjectMeta.Name,
		)
	}
}

func (h *Healer) runAWX(action *monitoring.AWXAction, alert *monitoring.Alert) {
	glog.Infof(
		"Running AWX action for alert '%s'",
		alert.ObjectMeta.Name,
	)
	glog.Infof("Project is '%s'", action.Project)
	glog.Infof("Job template is '%s'", action.JobTemplate)

	// Create the connection to the AWX server:
	connection, err := awx.NewConnectionBuilder().
		Url("https://tower.yellow/api/v2/").
		Proxy("http://server0.mad.redhat.com:3128").
		Username("admin").
		Password("redhat123").
		Insecure(true).
		Build()
	if err != nil {
		glog.Errorf("Can't create connection to AWX server: %s", err.Error())
		return
	}
	defer connection.Close()
}
