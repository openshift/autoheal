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
	"time"

	monitoring "github.com/openshift/autoheal/pkg/apis/monitoring/v1alpha1"
)

// Config is a read only view of the configuration of the auto-heal service.
//
type Config struct {
	awx        *AWXConfig
	throttling *ThrottlingConfig
	rules      []*monitoring.HealingRule
}

// ThrottlingConfig is a read only view of the section of the configuration that describes how to
// throttle the execution of healing rules.
//
type ThrottlingConfig struct {
	interval time.Duration
}

// AWX is a read only view of section of the configuration of the auto-heal service that describes
// how to connect to the AWX server, and how to launch jobs from templates.
//
type AWXConfig struct {
	address                string
	proxy                  string
	user                   string
	password               string
	insecure               bool
	ca                     []byte
	project                string
	jobStatusCheckInterval time.Duration
}

// AWX returns a read only view of the section of the configuration of the auto-heal service that
// describes how to connect to the AWX server, and how to launch jobs from templates.
//
func (c *Config) AWX() *AWXConfig {
	return c.awx
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
	return c.ca
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

// Throttling returns a read only view of the section of the configuration that describes how to
// throttle the execution of healing rules.
//
func (c *Config) Throttling() *ThrottlingConfig {
	return c.throttling
}

// Interval returns the throttling interval for the execution of the actions defined in the healing
// rules.
//
func (t *ThrottlingConfig) Interval() time.Duration {
	return t.interval
}

// Rules returns the list of healing rules defined in the configuration.
//
func (c *Config) Rules() []*monitoring.HealingRule {
	return c.rules
}
