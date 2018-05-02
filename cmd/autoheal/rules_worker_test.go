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
	"testing"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestPickRuleChange(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "empty-config.yml")
	healer, err := NewHealerBuilder().
		ConfigFile(file).
		Build()

	if err != nil {
		t.Errorf("Error building healer: %s", err)
	}

	change := &RuleChange{
		Type: watch.Added,
		Rule: &autoheal.HealingRule{
			ObjectMeta: meta.ObjectMeta{
				Name: "test-rule",
			},
			Labels: map[string]string{
				"mylabel": "myvalue",
			},
			AWXJob: &autoheal.AWXJobAction{
				Template: "test_template",
			},
		},
	}

	healer.rulesQueue.Add(change)

	if healer.pickRuleChange() != true {
		t.Errorf("Expected pickRuleChange to return true (i.e. there is a rule in ruleQueue) but got false")
	}
}

func TestProcessAddRuleChange(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "empty-config.yml")
	healer, err := NewHealerBuilder().
		ConfigFile(file).
		Build()

	if err != nil {
		t.Errorf("Error building healer: %s", err)
	}

	change := &RuleChange{
		Type: watch.Added,
		Rule: &autoheal.HealingRule{
			ObjectMeta: meta.ObjectMeta{
				Name: "test-rule",
			},
			Labels: map[string]string{
				"mylabel": "myvalue",
			},
			AWXJob: &autoheal.AWXJobAction{
				Template: "test_template",
			},
		},
	}

	healer.processRuleChange(change)
	_, ok := healer.rulesCache.Load(change.Rule.ObjectMeta.Name)
	if !ok {
		t.Errorf("Expected rulesCache to have a rule with key %s, instead there was no rule with that key", change.Rule.ObjectMeta.Name)
	}
}

func TestProcessDeletedRuleChange(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "empty-config.yml")
	healer, err := NewHealerBuilder().
		ConfigFile(file).
		Build()

	if err != nil {
		t.Errorf("Error building healer: %s", err)
	}

	change := &RuleChange{
		Type: watch.Deleted,
		Rule: &autoheal.HealingRule{
			ObjectMeta: meta.ObjectMeta{
				Name: "test-rule",
			},
			Labels: map[string]string{
				"mylabel": "myvalue",
			},
			AWXJob: &autoheal.AWXJobAction{
				Template: "test_template",
			},
		},
	}

	// Store dummy rule
	healer.rulesCache.Store(change.Rule.ObjectMeta.Name, change.Rule)
	// Delete dummy rule.
	healer.processRuleChange(change)
	_, ok := healer.rulesCache.Load(change.Rule.ObjectMeta.Name)
	if ok {
		t.Errorf("Expected rulesCache to delete rule with key %s, instead there is a rule with that key", change.Rule.ObjectMeta.Name)
	}
}

func TestProcessModifiedRuleChange(t *testing.T) {
	file := filepath.Join("..", "..", "testdata", "empty-config.yml")
	healer, err := NewHealerBuilder().
		ConfigFile(file).
		Build()

	if err != nil {
		t.Errorf("Error building healer: %s", err)
	}

	original := &autoheal.HealingRule{
		ObjectMeta: meta.ObjectMeta{
			Name:            "test-rule",
			ResourceVersion: "a",
		},
		Labels: map[string]string{
			"mylabel": "myvalue",
		},
		AWXJob: &autoheal.AWXJobAction{
			Template: "test_template",
		},
	}

	change := &RuleChange{
		Type: watch.Modified,
		Rule: &autoheal.HealingRule{
			ObjectMeta: meta.ObjectMeta{
				Name:            "test-rule",
				ResourceVersion: "b",
			},
			Labels: map[string]string{
				"mylabel": "my-changed-value",
			},
			AWXJob: &autoheal.AWXJobAction{
				Template: "test_template",
			},
		},
	}

	// Store dummy rule
	healer.rulesCache.Store("test-rule", original)
	// Modify dummy rule.
	healer.processRuleChange(change)
	val, _ := healer.rulesCache.Load("test-rule")
	rule := val.(*autoheal.HealingRule)
	if rule.Labels["mylabel"] != "my-changed-value" {
		t.Errorf("Expected rule label to have value %s, instead the value is %s", change.Rule.Labels["myvalue"], original.Labels["myvalue"])
	}
}
