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
	"github.com/openshift/autoheal/pkg/internal/data"
)

// EtcdConfig is a read only view of the section of the configuration that describes how to
// interact with etcd.
//
type EtcdConfig struct {
	endpoint string
}

// Endpoint returns the endpoint for etcd
//
func (t *EtcdConfig) Endpoint() string {
	return t.endpoint
}

func (t *EtcdConfig) merge(decoded *data.EtcdConfig) error {
	t.endpoint = decoded.Endpoint
	return nil
}
