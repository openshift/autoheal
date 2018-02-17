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
	"strings"

	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	labels "github.com/jhernand/openshift-monitoring/pkg/labels"
	mapping "github.com/jhernand/openshift-monitoring/pkg/mapping"
)

func mapAlert(input *alertmanager.Alert) (output *monitoring.Alert, err error) {
	// Create and populate the alert object:
	output = new(monitoring.Alert)

	// The alert should be created in the same namespace than the original alerting rule, or
	// else in the default namespace:
	namespace := input.Annotations["namespace"]
	if namespace == "" {
		namespace = "default"
	}
	output.ObjectMeta.Namespace = namespace

	// Initially the name of the alert is the name of the rule. It will be probably changed later
	// when trying to save it, to make it unique.
	name := input.Annotations["rule"]
	if name == "" {
		name = input.Labels["alertname"]
	}
	if name == "" {
		name = "unknown"
	}
	output.ObjectMeta.Name = strings.ToLower(name)

	// Copy the labels and annotations:
	mapping.CopyMap(input.Labels, &output.Status.Labels)
	mapping.CopyMap(input.Annotations, &output.Status.Annotations)

	// Calculate the hash and add it as a label:
	hash := hashAlert(output)
	if output.ObjectMeta.Labels == nil {
		output.ObjectMeta.Labels = make(map[string]string)
	}
	output.ObjectMeta.Labels[labels.Hash] = hash

	return
}
