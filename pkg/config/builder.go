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
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/apis/autoheal/v1alpha2"
)

// Builder contains the data and the methods needed to load the auto-heal service configuration.
//
type Builder struct {
	// The Kubernetes client that will be used to load Kubernetes objects:
	client kubernetes.Interface

	// The names of the configuration files, in the order that they should be loaded:
	files []string

	// The codec that will be used to convert the rules specified in the configuration file into the
	// types used internally.
	codec runtime.Codec
}

// NewBuilder creates an empty configuration loader.
//
func NewBuilder() *Builder {
	b := new(Builder)

	// Create and the codec that will be used to convert the rules specified into the configuration
	// file into the types used internally:
	scheme := runtime.NewScheme()
	autoheal.AddToScheme(scheme)
	v1alpha2.AddToScheme(scheme)
	b.codec = serializer.NewCodecFactory(scheme).LegacyCodec()

	return b
}

// Client sets the Kubernetes client that the configuration loader will use to load Kubernetes
// objects referenced from the configuration, like secrets or configuration maps. If this is not
// given then any reference to a Kubernetes object will cause an error when the configuration is
// loaded.
//
func (b *Builder) Client(client kubernetes.Interface) *Builder {
	b.client = client
	return b
}

// File adds the given file to the set of configuration files that will be loaded.
//
func (b *Builder) File(file string) *Builder {
	b.files = append(b.files, file)
	return b
}

// Files adds the given files to the set of configuration files that will be loaded.
//
func (b *Builder) Files(files []string) *Builder {
	if files != nil {
		for _, file := range files {
			b.files = append(b.files, file)
		}
	}
	return b
}

// Build loads the configuration files and returns the resulting configuration object.
//
func (b *Builder) Build() (c *Config, err error) {
	// Create an default configuration:
	c = &Config{
		awx: &AWXConfig{
			ca: new(bytes.Buffer),
			jobStatusCheckInterval: 5 * time.Minute,
			client:                 b.client,
		},
		throttling: &ThrottlingConfig{
			interval: 1 * time.Hour,
		},
		rules: &RulesConfig{
			codec: b.codec,
		},
		listener:      &eventListener{},
		files:         b.files,
		loadMutex:     &sync.Mutex{},
		listenerMutex: &sync.Mutex{},
	}

	// Do the initial load of the configuration files:
	err = c.load()
	if err != nil {
		return
	}

	// Start watching the configuration files:
	err = c.watch()
	if err != nil {
		return
	}

	return
}
