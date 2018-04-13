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

package v1alpha2

import (
	batch "k8s.io/api/batch/v1"
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
	meta.ObjectMeta `json:"metadata,omitempty"`

	// Labels is map containing the names of the labels and the regular expressions that they should
	// match in order to activate the rule.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is map containing the names of the annotations and the regular expressions that
	// they should match in order to activate the rule.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// AWXJob is the AWX job that will be executed when the rule is activated.
	// +optional
	AWXJob *AWXJobAction `json:"awxJob,omitempty"`

	// BatchJob is the batch job that will be executed when the rule is activated.
	// +optional
	BatchJob *batch.Job `json:"batchJob,omitempty"`
}

// AWXJobAction describes how to run an Ansible AWX job.
//
type AWXJobAction struct {
	// Template is the name of the AWX job template that will be launched.
	// +optional
	Template string `json:"template,omitempty"`

	// ExtraVars are the extra variables that will be passed to job.
	// +optional
	ExtraVars string `json:"extraVars,omitempty"`
}
