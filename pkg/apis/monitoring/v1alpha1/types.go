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

import (
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Alert is the description of an alert.
//
type Alert struct {
	meta.TypeMeta `json:",inline"`

	// Standard object metadata.
	// +optional
	meta.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// The status of the alert.
	// +optional
	Status AlertStatus `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
}

// AlertStatus is the status for an alert.
//
type AlertStatus struct {
	// Prometheus labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,1,rep,name=labels"`

	// Prometheus annotations.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,2,rep,name=annotations"`

	// Current state of the alert.
	// +optional
	Conditions []AlertCondition `json:"conditions,omitempty" protobuf:"bytes,3,rep,name=conditions"`
}

// AlertConditionType are the valid types of alert conditions.
//
type AlertConditionType string

// Valid types of alert conditions.
//
const (
	// The alert is firing.
	AlertFiring AlertConditionType = "Firing"

	// The alert is resolved.
	AlertResolved AlertConditionType = "Resolved"

	// The alert is being healed.
	AlertHealing AlertConditionType = "Healing"
)

// AlertCondition contains details for the current condition of the alert.
//
type AlertCondition struct {
	// The type of condition.
	Type AlertConditionType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`

	// Human readable message containing details of the condition.
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AlertList is a list of alerts.
//
type AlertList struct {
	meta.TypeMeta `json:",inline"`

	// Standard list metadata.
	// +optional
	meta.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of alerts.
	Items []Alert `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AlertRule is the description of an alerting rule.
//
type AlertingRule struct {
	meta.TypeMeta `json:",inline"`

	// Standard object metadata.
	// +optional
	meta.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the alerting rule.
	// +optional
	Spec AlertingRuleSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec`

	// Status represents the current information about an alenrting rule.
	// +optional
	Status AlertingRuleStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=spec`
}

// AlertingRuleSpec is the specification for an alerting rule.
//
type AlertingRuleSpec struct {
	// Group is the name of the group that the alerting rule should belong to.
	// +optional
	Group string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group`

	// Alert is the name of the rule that will be used inside Prometheus, and inside the alert
	// manager. If not specified will use the name of the object.
	// +optional
	Alert string `json:"alert,omitempty" protobuf:"bytes,2,opt,name=alert`

	// Expr is the expresion that defines the rule. The syntax is exactly the same used
	// to define alerting rules in Prometheus.
	Expr string `json:"expr,omitempty" protobuf:"bytes,3,opt,name=expr`

	// For indicates how long Prometheus will wait till the expression evaluates to true
	// till the alert is actually fired.
	// +optional
	For string `json:"for,omitempty" protobuf:"bytes,4,opt,name=spec`

	// Prometheus labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,5,rep,name=labels"`

	// Prometheus annotations.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,6,rep,name=annotations"`
}

// AlertingRuleStatus is the status for an alerting rule.
//
type AlertingRuleStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AlertingRuleList is a list of alerting rules.
//
type AlertingRuleList struct {
	meta.TypeMeta `json:",inline"`

	// Standard list metadata.
	// +optional
	meta.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of alerting rules.
	Items []AlertingRule `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HealingRule is the description of an healing rule.
//
type HealingRule struct {
	meta.TypeMeta `json:",inline"`

	// Standard object metadata.
	// +optional
	meta.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the healing rule.
	// +optional
	Spec HealingRuleSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec`

	// Status represents the current information about an healing rule.
	// +optional
	Status HealingRuleStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=spec`
}

// HealingRuleSpec is the specification for a healing rule.
//
type HealingRuleSpec struct {
	// Conditions is the set of conditions that should be true for the rule to be activated.
	// +optional
	Conditions []HealingCondition `json:"conditions" protobuf:"bytes,1,rep,name=conditions`

	// Actions is the set of healing actions that will be executed when the rule is activated.
	// +optional
	Actions []HealingAction `json:"actions" protobuf:"bytes,2,rep,name=conditions`
}

// HealingCondition represents a condition for the activation of a healing rule.
//
type HealingCondition struct {
	// Alert is the name of an alert that has to exist for this condition to be true.
	// +optional
	Alert string `json:"alert,omitempty" protobuf:"bytes,1,opt,name=alert`
}

// HealingAction represents an action that will be performed when a healing rule is activated.
//
type HealingAction struct {
	// AWXJob is used when the healing action is implemented by an Ansible AWX job.
	// +optional
	AWXJob *AWXJobAction `json:"awxJob,omitempty" protobuf:"bytes,1,opt,name=awxJob`

	// BatchJob is used when the healing action is implemented by a Kubernetes batch job.
	// +optional
	BatchJob *batch.Job `json:"batchJob,omitempty" protobuf:"bytes,2,opt,name=batchJob`

	// AnsiblePlaybook is used when the healing action is implemented by an Ansible playbook.
	// +optional
	AnsiblePlaybook *AnsiblePlaybookAction `json:"ansiblePlaybook,omitempty" protobuf:"bytes,3,opt,name=ansiblePlaybook`
}

// AWXJobAction describes how to run an Ansible AWX job.
//
type AWXJobAction struct {
	// Address is the complete URL used to access the API of the AWX server.
	// +optional
	Address string `json:"address,omitempty" protobuf:"bytes,1,opt,name=address`

	// Proxy is the address of the proxy server used to access the API of the AWX server.
	// +optional
	Proxy string `json:"proxy,omitempty" protobuf:"bytes,2,opt,name=proxy`

	// SecretRef is the reference (name, and optionally namespace) of the secret that contains the
	// user name and password used to access the AWX API.
	// +optional
	SecretRef *core.SecretReference `json:"secretRef,omitempty" protobuf:"bytes,3,opt,name=secretRef`

	// Project is the name of the AWX project that contains the job template.
	// +optional
	Project string `json:"project,omitempty" protobuf:"bytes,4,opt,name=project`

	// Template is the name of the AWX job template that will be launched.
	// +optional
	Template string `json:"template,omitempty" protobuf:"bytes,5,opt,name=template`

	// ExtraVars are the extra variables that will be passed to job.
	// +optional
	ExtraVars string `json:"extraVars,omitempty" protobuf:"bytes,5,opt,name=extraVars`
}

// AnsiblePlaybookAction describes ho to run an Ansible playbook.
//
type AnsiblePlaybookAction struct {
	// Playbook is the complete text of the playbook.
	// +optional
	Playbook string `json:"playbook,omitempty" protobuf:"bytes,1,opt,name=playbook`

	// Inventory is the complete text of the inventory.
	// +optional
	Inventory string `json:"inventory,omitempty" protobuf:"bytes,2,opt,name=inventory`

	// SecretRef is the reference (name, and optionally namespace) of the secret that contains the
	// SSH private key that Ansible will use to access the hosts.
	// +optional
	SecretRef *core.SecretReference `json:"secretRef,omitempty" protobuf:"bytes,3,opt,name=secretRef`

	// ExtraVars are the extra variables that will be passed to Ansible.
	// +optional
	ExtraVars string `json:"extraVars,omitempty" protobuf:"bytes,4,opt,name=extraVars`
}

// HealingRuleStatus is the status for an alerting rule.
//
type HealingRuleStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HealingRuleList is a list of alerting rules.
//
type HealingRuleList struct {
	meta.TypeMeta `json:",inline"`

	// Standard list metadata.
	// +optional
	meta.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of alerting rules.
	Items []HealingRule `json:"items" protobuf:"bytes,2,rep,name=items"`
}
