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

package prometheus

import (
	"time"
)

// AlertingConfig represents the Prometheus alerting configuration.
//
type AlertingConfig struct {
	Groups []*AlertingRuleGroup `yaml:"groups,omitempty"`
}

// AlertingRuleGroups represents a group of Prometheus alerting rules.
//
type AlertingRuleGroup struct {
	Name  string          `yaml:"name,omitempty"`
	Rules []*AlertingRule `yaml:"rules,omitempty"`
}

// AlertingRule represents a Prometheus alerting rule.
//
type AlertingRule struct {
	Alert       string            `yaml:"alert,omitempty"`
	Expr        string            `yaml:"expr,omitempty"`
	For         time.Duration     `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
