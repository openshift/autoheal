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
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
)

func (h *Healer) runRulesWorker() {
	for h.pickRuleChange() {
		// Nothing.
	}
}

func (h *Healer) pickRuleChange() bool {
	// Get the next item and end the work loop if asked to stop:
	item, stop := h.rulesQueue.Get()
	if stop {
		return false
	}

	// Process the item and make sure to always tell the queue that we are done with this item:
	err := func(item interface{}) error {
		h.rulesQueue.Done(item)

		// Check that the item we got from the queue is really a change, and discard it otherwise:
		change, ok := item.(*RuleChange)
		if !ok {
			h.rulesQueue.Forget(item)
		}

		// Process and then forget the change:
		err := h.processRuleChange(change)
		if err != nil {
			return err
		}
		h.rulesQueue.Forget(change)

		return nil
	}(item)
	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (h *Healer) processRuleChange(change *RuleChange) error {
	switch change.Type {
	case watch.Added:
		return h.processAddedRule(change.Rule)
	case watch.Modified:
		return h.processAddedRule(change.Rule)
	case watch.Deleted:
		return h.processDeletedRule(change.Rule)
	}
	return nil
}

func (h *Healer) processAddedRule(rule *autoheal.HealingRule) error {
	value, ok := h.rulesCache.Load(rule.ObjectMeta.Name)
	if !ok {
		h.rulesCache.Store(rule.ObjectMeta.Name, rule)
		glog.Infof("Rule '%s' was added", rule.ObjectMeta.Name)
	} else {
		existing := value.(*autoheal.HealingRule)
		if rule.ObjectMeta.ResourceVersion != existing.ObjectMeta.ResourceVersion {
			h.rulesCache.Store(rule.ObjectMeta.Name, rule)
			glog.Infof("Rule '%s' was updated", rule.ObjectMeta.Name)
		}
	}
	return nil
}

func (h *Healer) processDeletedRule(rule *autoheal.HealingRule) error {
	_, ok := h.rulesCache.Load(rule.ObjectMeta.Name)
	if ok {
		h.rulesCache.Delete(rule.ObjectMeta.Name)
		glog.Infof("Rule '%s' was deleted", rule.ObjectMeta.Name)
	}
	return nil
}
