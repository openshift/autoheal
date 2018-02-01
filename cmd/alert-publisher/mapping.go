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
	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	mapping "github.com/jhernand/openshift-monitoring/pkg/mapping"
)

func mapAlert(input *alertmanager.Alert) (output *monitoring.Alert, err error) {
	// Create and populate the alert object:
	output = new(monitoring.Alert)

	// Map the basic data:
	output.ObjectMeta.Namespace = input.Annotations["namespace"]
	output.ObjectMeta.Name = input.Labels["alertname"]

	// Copy the labels and annotations:
	mapping.CopyMap(input.Labels, &output.Status.Labels)
	mapping.CopyMap(input.Annotations, &output.Status.Annotations)

	return
}
