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
	// Delimiters indicates the delimiters that will be used in the text templates instead of the
	// default {{ and }} used in Go templates. This is specially convenient when the text of the
	// action contains an Ansible playbook, as the Go delimeters conflict with the Jinja2
	// delimiters.
	// +optional
	Delimiters *Delimiters `json:"delimiters,omitempty" protobuf:"bytes,1,opt,name=delimiters`

	// AWXJob is used when the healing action is implemented by an Ansible AWX job.
	// +optional
	AWXJob *AWXJobAction `json:"awxJob,omitempty" protobuf:"bytes,2,opt,name=awxJob`

	// BatchJob is used when the healing action is implemented by a Kubernetes batch job.
	// +optional
	BatchJob *batch.Job `json:"batchJob,omitempty" protobuf:"bytes,3,opt,name=batchJob`

	// AnsiblePlaybook is used when the healing action is implemented by an Ansible playbook.
	// +optional
	AnsiblePlaybook *AnsiblePlaybookAction `json:"ansiblePlaybook,omitempty" protobuf:"bytes,4,opt,name=ansiblePlaybook`
}

// Delimiters indicates the delimiters used to mark expressions inside text templates.
//
type Delimiters struct {
	Left  string `json:"left,omitempty" protobuf:"bytes,1,opt,name=left`
	Right string `json:"right,omitempty" protobuf:"bytes,2,opt,name=right`
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
