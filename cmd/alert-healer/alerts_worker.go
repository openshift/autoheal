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
	"regexp"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/runtime"

	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
)

func (h *Healer) runAlertsWorker() {
	for h.pickAlert() {
		// Nothing.
	}
}

func (h *Healer) pickAlert() bool {
	// Get the next item and end the work loop if asked to stop:
	item, stop := h.alertsQueue.Get()
	if stop {
		return false
	}

	// Process the item and make sure to always tell the queue that we are done with this item:
	err := func(item interface{}) error {
		h.alertsQueue.Done(item)

		// Check that the item we got from the queue is really an alert, and discard it otherwise:
		alert, ok := item.(*alertmanager.Alert)
		if !ok {
			h.alertsQueue.Forget(item)
		}

		// Process and then forget the alert:
		err := h.processAlert(alert)
		if err != nil {
			return err
		}
		h.alertsQueue.Forget(alert)

		return nil
	}(item)
	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (h *Healer) processAlert(alert *alertmanager.Alert) error {
	switch alert.Status {
	case alertmanager.AlertStatusFiring:
		return h.startHealing(alert)
	case alertmanager.AlertStatusResolved:
		return h.cancelHealing(alert)
	default:
		glog.Warningf(
			"Unknnown status '%s' reported by alert manager, will ignore it",
			alert.Status,
		)
		return nil
	}
}

// startHealing starts the healing process for the given alert.
//
func (h *Healer) startHealing(alert *alertmanager.Alert) error {
	// Find the rules that are activated for the alert:
	activated := make([]*monitoring.HealingRule, 0)
	h.rulesCache.Range(func(_, value interface{}) bool {
		rule := value.(*monitoring.HealingRule)
		if h.checkConditions(rule, alert) {
			glog.Infof(
				"Healing rule '%s' matches alert '%s'",
				rule.ObjectMeta.Name,
				alert.Name(),
			)
			activated = append(activated, rule)
		}
		return true
	})
	if len(activated) == 0 {
		glog.Infof("No healing rule matches alert '%s'", alert.Name())
		return nil
	}

	// Execute the actions of the activated rules:
	for _, rule := range activated {
		err := h.runActions(rule, alert)
		if err != nil {
			return err
		}
	}

	return nil
}

// cancelHealing cancels the healing process for the given alert.
//
func (h *Healer) cancelHealing(alert *alertmanager.Alert) error {
	return nil
}

func (h *Healer) checkConditions(rule *monitoring.HealingRule, alert *alertmanager.Alert) bool {
	glog.Infof(
		"Checking conditions of rule '%s' for alert '%s'",
		rule.ObjectMeta.Name,
		alert.Name(),
	)
	if rule.Spec.Conditions != nil && len(rule.Spec.Conditions) > 0 {
		for i := 0; i < len(rule.Spec.Conditions); i++ {
			if !h.checkCondition(&rule.Spec.Conditions[i], alert) {
				return false
			}
		}
	}
	return true
}

func (h *Healer) checkCondition(condition *monitoring.HealingCondition, alert *alertmanager.Alert) bool {
	matched, err := regexp.MatchString(condition.Alert, alert.Name())
	if err != nil {
		glog.Errorf(
			"Error while checking if alert name '%s' matches pattern '%s': %s",
			alert.Name(),
			condition.Alert,
			err.Error(),
		)
		matched = false
	}
	return matched
}

func (h *Healer) runActions(rule *monitoring.HealingRule, alert *alertmanager.Alert) error {
	if rule.Spec.Actions != nil && len(rule.Spec.Actions) > 0 {
		glog.Infof(
			"Running actions of healing rule '%s' for alert '%s'",
			rule.ObjectMeta.Name,
			alert.Name(),
		)
		for i := 0; i < len(rule.Spec.Actions); i++ {
			err := h.runAction(rule, &rule.Spec.Actions[i], alert)
			if err != nil {
				return err
			}
		}
	} else {
		glog.Warningf(
			"Healing rule '%s' has no actions, will have no effect on alert '%s'",
			rule.ObjectMeta.Name,
			alert.Name(),
		)
	}

	return nil
}

func (h *Healer) runAction(rule *monitoring.HealingRule, action *monitoring.HealingAction, alert *alertmanager.Alert) error {
	// Make a copy of the action to ensure that the original, which is stored in the rules cache, it
	// isn't modified during the rest of the process:
	action = action.DeepCopy()

	// Remove the template configuration from the copy, as we don't want to process the delimiters
	// themselves as templates, would generate errors.
	var left, right string
	if action.Delimiters != nil {
		left = action.Delimiters.Left
		right = action.Delimiters.Right
	}
	action.Delimiters = nil

	// Process the templates inside the the action:
	template, err := NewObjectTemplateBuilder().
		Delimiters(left, right).
		Variable("alert", ".").
		Variable("labels", ".Labels").
		Variable("annotations", ".Annotations").
		Build()
	if err != nil {
		return err
	}
	err = template.Process(action, alert)
	if err != nil {
		return err
	}

	// Decide which kind of action to run, and run it:
	if action.AWXJob != nil {
		return h.runAWXJob(rule, action.AWXJob, alert)
	} else if action.BatchJob != nil {
		return h.runBatchJob(rule, action.BatchJob, alert)
	} else if action.AnsiblePlaybook != nil {
		err = template.Process(action.AnsiblePlaybook, alert)
		if err != nil {
			return err
		}
		return h.runAnsiblePlaybook(rule, action.AnsiblePlaybook, alert)
	} else {
		glog.Warningf(
			"There are no action details, rule '%s' will have no effect on alert '%s'",
			rule.ObjectMeta.Name,
			alert.Name(),
		)
	}
	return nil
}
