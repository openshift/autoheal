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
	"path/filepath"
	"reflect"
	"testing"

	"github.com/openshift/autoheal/pkg/alertmanager"
	"github.com/openshift/autoheal/pkg/apis/autoheal"
	batch "k8s.io/api/batch/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPickAlert(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "empty-config.yml")
	healer, err := NewHealerBuilder().
		ConfigFile(file).
		Build()

	if err != nil {
		t.Error(err)
	}

	alert := &alertmanager.Alert{
		Status: "firing",
		Labels: map[string]string{
			"mylabel": "myvalue",
		},
	}

	healer.alertsQueue.Add(alert)

	// one alert in queue.
	if healer.pickAlert() != true {
		t.Errorf("Expected pickAlert to return true (i.e. there is an alert in alertQueue) but got false")
	}
}

func TestStartHealingAWXJob(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "empty-config.yml")
	healer, err := NewHealerBuilder().
		ConfigFile(file).
		Build()

	if err != nil {
		t.Error(err)
	}

	actionRunner := FakeActionRunner{
		RuleAlertMap: make(map[string]*alertmanager.Alert),
	}

	healer.actionRunners[ActionRunnerTypeAWX] = actionRunner

	alert := &alertmanager.Alert{
		Status: "firing",
		Labels: map[string]string{
			"mylabel": "myvalue",
		},
	}

	rule := &autoheal.HealingRule{
		ObjectMeta: meta.ObjectMeta{
			Name: "test-rule",
		},
		Labels: map[string]string{
			"mylabel": "myvalue",
		},
		AWXJob: &autoheal.AWXJobAction{
			Template: "Test AWX JOB",
		},
	}

	// Add the rule to rulesCache
	healer.rulesCache.Store(rule.ObjectMeta.Name, rule)

	healer.startHealing(alert)

	expected := map[string]*alertmanager.Alert{
		rule.ObjectMeta.Name: alert,
	}

	if reflect.DeepEqual(expected, actionRunner.RuleAlertMap) != true {
		t.Errorf("Expected action runner map to be equal to %+v, instead got %+v",
			expected,
			actionRunner.RuleAlertMap)
	}
}

func TestStartHealingBatchJob(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "empty-config.yml")
	healer, err := NewHealerBuilder().
		ConfigFile(file).
		Build()

	if err != nil {
		t.Error(err)
	}

	actionRunner := FakeActionRunner{
		RuleAlertMap: make(map[string]*alertmanager.Alert),
	}

	healer.actionRunners[ActionRunnerTypeBatch] = actionRunner

	alert := &alertmanager.Alert{
		Status: "firing",
		Labels: map[string]string{
			"mylabel": "other-value",
		},
	}

	rule := &autoheal.HealingRule{
		ObjectMeta: meta.ObjectMeta{
			Name: "test-batch-rule",
		},
		Labels: map[string]string{
			"mylabel": "other-value",
		},
		BatchJob: &batch.Job{
			ObjectMeta: meta.ObjectMeta{
				Namespace: "default",
				Name:      "hello",
			},
		},
	}

	healer.rulesCache.Store(rule.ObjectMeta.Name, rule)

	healer.startHealing(alert)

	expected := map[string]*alertmanager.Alert{
		rule.ObjectMeta.Name: alert,
	}

	if reflect.DeepEqual(expected, actionRunner.RuleAlertMap) != true {
		t.Errorf("Expected action runner map to be equal to %+v, instead got %+v",
			expected,
			actionRunner.RuleAlertMap)
	}
}
