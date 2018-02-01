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
	"sort"
	"strings"

	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	mapping "github.com/jhernand/openshift-monitoring/pkg/mapping"
	prometheus "github.com/jhernand/openshift-monitoring/pkg/prometheus"
	yaml "gopkg.in/yaml.v2"
)

// makeYaml takes a list of alerting rules and generates the Prometheus configuration file
// containing them.
//
func makeYaml(input []*monitoring.AlertingRule) (data []byte, err error) {
	output, err := mapConfig(input)
	if err != nil {
		return
	}
	data, err = yaml.Marshal(output)
	if err != nil {
		return
	}

	return
}

func mapConfig(input []*monitoring.AlertingRule) (output *prometheus.AlertingConfig, err error) {
	// Create an empty configuration:
	output = new(prometheus.AlertingConfig)

	// Map the groups:
	output.Groups, err = mapGroups(input)
	if err != nil {
		return
	}

	return
}

func mapGroups(input []*monitoring.AlertingRule) (output []*prometheus.AlertingRuleGroup, err error) {
	// Group the rules by group name:
	groups := make(map[string][]*monitoring.AlertingRule)
	for _, rule := range input {
		name := rule.Spec.Group
		if name == "" {
			name = "default"
		}
		group := groups[name]
		if group == nil {
			group = make([]*monitoring.AlertingRule, 0)
		}
		group = append(group, rule)
		groups[name] = group
	}

	// Map the groups:
	output = make([]*prometheus.AlertingRuleGroup, len(groups))
	i := 0
	for name, group := range groups {
		output[i], err = mapGroup(name, group)
		if err != nil {
		}
		i++
	}

	// We need to sort the groups by name, to make sure that the result will always be predictable
	// regardless of the order of the input.
	less := func(i, j int) bool {
		left := output[i].Name
		right := output[j].Name
		return strings.Compare(left, right) < 0
	}
	sort.Slice(output, less)

	return
}

func mapGroup(name string, input []*monitoring.AlertingRule) (output *prometheus.AlertingRuleGroup, err error) {
	// Create an empty group:
	output = new(prometheus.AlertingRuleGroup)

	// Map the basic attributes:
	output.Name = name

	// Map the rules:
	output.Rules = make([]*prometheus.AlertingRule, len(input))
	for i, rule := range input {
		output.Rules[i], err = mapRule(rule)
		if err != nil {
			return
		}
	}

	// We need to sort the rules by name, to make sure that the result will always be predictable
	// regardless of the order of the input.
	less := func(i, j int) bool {
		left := output.Rules[i].Alert
		right := output.Rules[j].Alert
		return strings.Compare(left, right) < 0
	}
	sort.Slice(output.Rules, less)

	return
}

func mapRule(input *monitoring.AlertingRule) (output *prometheus.AlertingRule, err error) {
	// Create an empty rule:
	output = new(prometheus.AlertingRule)

	// Map the basic attributes:
	output.Alert = input.Spec.Alert
	if output.Alert == "" {
		output.Alert = input.ObjectMeta.Name
	}
	output.Expr = input.Spec.Expr
	output.For = input.Spec.For

	// Map the labels and annotations. Note that there is no need (or way) to sort these as the YAML
	// library sorts them, using the map keys.
	mapping.CopyMap(input.Spec.Labels, &output.Labels)
	mapping.CopyMap(input.Spec.Annotations, &output.Annotations)

	// Add an annotation with the Kubernetes namespace, so that it is propagated to the alert that
	// will be eventually generated. Otherwise when the alert is translated into a Kubernetes object
	// there is no way to reliably set the namespace.
	output.Annotations["namespace"] = input.ObjectMeta.Namespace

	return
}
