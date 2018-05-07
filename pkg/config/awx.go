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

// Package config contains types and functions used to load the service configuration.
//
package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/autoheal/pkg/internal/data"
)

// AWX is a read only view of section of the configuration of the auto-heal service that describes
// how to connect to the AWX server, and how to launch jobs from templates.
//
type AWXConfig struct {
	address                string
	proxy                  string
	user                   string
	password               string
	insecure               bool
	ca                     *bytes.Buffer
	project                string
	jobStatusCheckInterval time.Duration

	// The Kubernetes client that will be used to load Kubernetes objects:
	client kubernetes.Interface
}

// Address returns the complete address of the API of the AWX server, including the /api suffix,
// but not the /v1 or /v2 suffixes.
//
func (c *AWXConfig) Address() string {
	return c.address
}

// Proxy returns the complete address of the proxy server that the auto-heal service should use to
// connect to the API of the AWX server. The format is an URL, where only the host name and the port
// number are relevant:
//
//	http://myproxy.example.com:3128
//
// An empty string means that no proxy should be used.
//
func (c *AWXConfig) Proxy() string {
	return c.proxy
}

// User returns the name of the user that the auto-heal service will use to connect to the AWX
// server.
//
func (c *AWXConfig) User() string {
	return c.user
}

// Password returns the password of the user that the auto-heal service will use to connect to
// the AWX server.
//
func (c *AWXConfig) Password() string {
	return c.password
}

// CA returns the PEM encoded certificates of the authorities that should be trusted when checking
// the TLS certificate presented by the AWX server. If not provided the system cert pool will be used.
//
func (c *AWXConfig) CA() []byte {
	if c.ca == nil {
		return nil
	}
	return c.ca.Bytes()
}

// Project returns the name of the AWX project that contains the auto-heal job templates.
//
func (c *AWXConfig) Project() string {
	return c.project
}

// Whether to use insecure connection to connect to AWX.
//
func (c *AWXConfig) Insecure() bool {
	return c.insecure
}

// Return the duration of how often the active AWX jobs status is checked
//
func (c *AWXConfig) JobStatusCheckInterval() time.Duration {
	return c.jobStatusCheckInterval
}

func (a *AWXConfig) merge(decoded *data.AWXConfig) error {
	// Merge the server address and proxy:
	if decoded.Address != "" {
		a.address = decoded.Address
	}
	if decoded.Proxy != "" {
		a.proxy = decoded.Proxy
	}

	// Merge the credentials:
	if decoded.Credentials != nil {
		err := a.mergeAWXCredentials(decoded.Credentials)
		if err != nil {
			return err
		}
	}
	if decoded.CredentialsRef != nil {
		err := a.mergeAWXCredentialsSecret(decoded.CredentialsRef)
		if err != nil {
			return err
		}
	}

	// Merge the TLS details:
	if decoded.TLS != nil {
		err := a.mergeAWXTLS(decoded.TLS)
		if err != nil {
			return err
		}
	}
	if decoded.TLSRef != nil {
		err := a.mergeAWXTLSSecret(decoded.TLSRef)
		if err != nil {
			return err
		}
	}

	// Merge insecure setting:
	a.insecure = decoded.Insecure

	// Merge the project:
	if decoded.Project != "" {
		a.project = decoded.Project
	}

	// Merge the jobStatusCheckInterval
	if decoded.JobStatusCheckInterval != "" {
		interval, err := time.ParseDuration(decoded.JobStatusCheckInterval)
		if err != nil {
			return err
		}
		a.jobStatusCheckInterval = interval
	}

	return nil
}

func (a *AWXConfig) mergeAWXCredentials(credentials *data.AWXCredentialsConfig) error {
	if credentials.Username != "" {
		a.user = credentials.Username
	}
	if credentials.Password != "" {
		a.password = credentials.Password
	}
	return nil
}

func (a *AWXConfig) mergeAWXCredentialsSecret(reference *core.SecretReference) error {
	secret, err := a.loadSecret(reference)
	if err != nil {
		return err
	}
	if secret.Data != nil {
		var value []byte
		var ok bool
		value, ok = secret.Data[core.BasicAuthUsernameKey]
		if ok {
			a.user = string(value)
		}
		value, ok = secret.Data[core.BasicAuthPasswordKey]
		if ok {
			a.password = string(value)
		}
	}
	return nil
}

func (a *AWXConfig) mergeAWXTLS(tls *data.TLSConfig) error {
	if tls.CACerts != "" {
		a.ca.WriteString(tls.CACerts)
	}
	if tls.CAFile != "" {
		caCerts, err := ioutil.ReadFile(tls.CAFile)
		if err != nil {
			return err
		}
		a.ca.Write(caCerts)
	}
	return nil
}

func (a *AWXConfig) mergeAWXTLSSecret(reference *core.SecretReference) error {
	secret, err := a.loadSecret(reference)
	if err != nil {
		return err
	}
	if secret.Data != nil {
		var value []byte
		var ok bool
		value, ok = secret.Data[core.ServiceAccountRootCAKey]
		if ok {
			a.ca.Write(value)
		}
	}
	return nil
}

func (a *AWXConfig) loadSecret(reference *core.SecretReference) (secret *core.Secret, err error) {
	// Both the name and the namespace are mandatory:
	if reference.Name == "" {
		err = fmt.Errorf("The name of the secret is mandatory, but it hasn't been specified")
		return
	}
	if reference.Namespace == "" {
		err = fmt.Errorf("The namespace of the secret is mandatory, but it hasn't been specified")
		return
	}

	// Check that we have a client to use the Kubernetes API:
	if a.client == nil {
		err = fmt.Errorf(
			"Can't load secret '%s' from namespace '%s' because there is no connection to the Kubernetes API",
			reference.Name,
			reference.Namespace,
		)
		return
	}

	// Try to retrieve the secret:
	resource := a.client.CoreV1().Secrets(reference.Namespace)
	secret, err = resource.Get(reference.Name, meta.GetOptions{})
	if err != nil {
		err = fmt.Errorf(
			"Can't load secret '%s' from namespace '%s': %s",
			reference.Name,
			reference.Namespace,
			err,
		)
		return
	}

	return
}
