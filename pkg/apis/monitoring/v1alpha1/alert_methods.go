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

package v1alpha1

// GetCondition returns the first condition of the given kind, or nil if the alert doesn't have a
// condition of that kind.
//
func (a *Alert) GetCondition(kind AlertConditionType) *AlertCondition {
	if a.Status.Conditions == nil {
		return nil
	}
	for _, condition := range a.Status.Conditions {
		if condition.Type == kind {
			return &condition
		}
	}
	return nil
}

// HasCondition returns true if the alert has any condition of the given kind.
//
func (a *Alert) HasCondition(kind AlertConditionType) bool {
	return a.GetCondition(kind) != nil
}

// AddCondition adds a condition of the given type.
//
func (a *Alert) AddCondition(kind AlertConditionType) {
	condition := a.GetCondition(kind)
	if condition == nil {
		a.Status.Conditions = append(a.Status.Conditions, AlertCondition{
			Type: kind,
		})
	}
}

// AddConditionMessage adds a condition of the given type and with the given message.
//
func (a *Alert) AddConditionMessage(kind AlertConditionType, message string) {
	condition := a.GetCondition(kind)
	if condition == nil {
		a.Status.Conditions = append(a.Status.Conditions, AlertCondition{
			Type:    kind,
			Message: message,
		})
	} else {
		condition.Message = message
	}
}

// DeleteCondition deletes all the conditions of the given type.
//
func (a *Alert) DeleteCondition(kind AlertConditionType) {
	if a.Status.Conditions != nil {
		var filtered []AlertCondition
		for _, condition := range a.Status.Conditions {
			if condition.Type != kind {
				filtered = append(filtered, condition)
			}
		}
		a.Status.Conditions = filtered
	}
}
