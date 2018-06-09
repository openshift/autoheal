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
)

// Config is used to marshal and unmarshal the main configuration of the auto-heal service.
//
type Config struct {
	// AWX contains the details to connect to the default AWX server.
	AWX *AWXConfig `json:"awx,omitempty"`

	// Throttling contains the healing rule execution throttling details.
	Throttling *ThrottlingConfig

	// EtcdConfig contains details about etcd.
	Etcd *EtcdConfig

	// The list of healing rules. Note that we use here an interface because we don't know in
	// advance what version of the rule type will be used in the configuration file. So we accept
	// any thing and we will try to convert them to the internal unversioned rule type using the
	// standard Kubernetes API mechanisms.
	Rules []interface{} `json:"rules,omitempty"`
}

// AWXConfig contains the details used by the auto-heal service to connect to the AWX server and
// launch jobs from templates.
//
type AWXConfig struct {
	// URL is the complete URL used to access the API of the AWX server.
	Address string `json:"address,omitempty"`

	// Proxy is the address of the proxy server used to access the API of the AWX server.
	Proxy string `json:"proxy,omitempty"`

	// Credentials contains the user name and password.
	Credentials *AWXCredentialsConfig `json:"credentials,omitempty"`

	// CredentialsRef is the reference (name, and optionally namespace) of the secret that contains
	// the user name and password used to access the AWX API.
	CredentialsRef *core.SecretReference `json:"credentialsRef,omitempty"`

	// TLS contains the TLS configuration.
	TLS *TLSConfig `json:"tls,omitempty"`

	// TLSRef is the reference (name, and optionally namespace) of the secret that contains the TLS
	// certificates and keys used to access the AWX API.
	TLSRef *core.SecretReference `json:"tlsRef,omitempty"`

	// Whether to use an insecure connection to connect to AWX.
	Insecure bool `json:"insecure,omitempty"`

	// Project is the name of the AWX project that contains the job templates.
	Project string `json:"project,omitempty"`

	// JobStatusCheckInterval determines how often to check AWX active jobs status
	JobStatusCheckInterval string `json:"jobStatusCheckInterval,omitempty"`
}

// AWXCredentialsConfig contains the credentials used to connect to the AWX server.
//
type AWXCredentialsConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// TLSConfig contains the TLS configuration.
//
type TLSConfig struct {
	CACerts string `json:"caCerts,omitempty"`
	CAFile  string `json:"caFile,omitempty"`
}

// ThrottlingConfig is used to mardhal and unmarshal the healing rule exeuction throttling
// configuration.
//
type ThrottlingConfig struct {
	Interval string `json:"interval,omitempty"`
}

// EtcdConfig is etcd related configurations
type EtcdConfig struct {
	Endpoint string `json:"endpoint,omitempty"`
}
