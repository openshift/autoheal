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

// This file contains the types used to marshal and unmarshall the types that contain the
// configuration of the auto-heal service.

package data

import (
	core "k8s.io/api/core/v1"

	monitoring "github.com/openshift/autoheal/pkg/apis/monitoring/v1alpha1"
)

// Config is used to marshal and unmarshal the main configuration of the auto-heal service.
//
type Config struct {
	// AWX contains the details to connect to the default AWX server.
	AWX *AWXConfig `json:"awx,omitempty"`

	// The list of healing rules.
	Rules []*monitoring.HealingRule `json:"rules,omitempty"`
}

// AWXConfig contains the details used by the auto-heal service to connect to the AWX server and
// launch jobs from templates.
//
type AWXConfig struct {
	// URL is the complete URL used to access the API of the AWX server.
	Address string `json:"address,omitempty"`

	// Proxy is the address of the proxy server used to access the API of the AWX server.
	Proxy string `json:"proxy,omitempty"`

	// CredentialsRef is the reference (name, and optionally namespace) of the secret that contains
	// the user name and password used to access the AWX API.
	CredentialsRef *core.SecretReference `json:"credentialsRef,omitempty"`

	// TLSRef is the reference (name, and optionally namespace) of the secret that contains the TLS
	// certificates and keys used to access the AWX API.
	TLSRef *core.SecretReference `json:"tlsRef,omitempty"`

	// Project is the name of the AWX project that contains the job templates.
	Project string `json:"project,omitempty"`
}
