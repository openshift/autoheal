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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/apis/autoheal/v1alpha2"
	"github.com/openshift/autoheal/pkg/internal/data"
)

// Loader contains the data and the methods needed to load the auto-heal service configuration.
//
type Loader struct {
	// The Kubernetes client that will be used to load Kubernetes objects:
	client kubernetes.Interface

	// The names of the configuration files, in the order that they should be loaded:
	files []string

	// The configuration object that is being populated:
	config *Config

	// The codec that will be used to convert the rules specified in the configuration file into the
	// types used internally.
	codec runtime.Codec
}

// NewLoader creates an empty configuration loader.
//
func NewLoader() *Loader {
	l := new(Loader)

	// Create and the codec that will be used to convert the rules specified into the configuration
	// file into the types used internally:
	scheme := runtime.NewScheme()
	autoheal.AddToScheme(scheme)
	v1alpha2.AddToScheme(scheme)
	l.codec = serializer.NewCodecFactory(scheme).LegacyCodec()

	return l
}

// Client sets the Kubernetes client that the configuration loader will use to load Kubernetes
// objects referenced from the configuration, like secrets or configuration maps. If this is not
// given then any reference to a Kubernetes object will cause an error when the configuration is
// loaded.
//
func (l *Loader) Client(client kubernetes.Interface) *Loader {
	l.client = client
	return l
}

// File adds the given file to the set of configuration files that will be loaded.
//
func (l *Loader) File(file string) *Loader {
	l.files = append(l.files, file)
	return l
}

// Files adds the given files to the set of configuration files that will be loaded.
//
func (l *Loader) Files(files []string) *Loader {
	if files != nil {
		for _, file := range files {
			l.files = append(l.files, file)
		}
	}
	return l
}

// Load loads the configuration files and returns the resulting configuration object.
//
func (l *Loader) Load() (config *Config, err error) {
	// Create an default configuration:
	l.config = &Config{
		awx: &AWXConfig{
			ca: new(bytes.Buffer),
			jobStatusCheckInterval: 5 * time.Minute,
		},
		throttling: &ThrottlingConfig{
			interval: 1 * time.Hour,
		},
	}

	// Merge the contents of the files into the empty configuration:
	for _, file := range l.files {
		var info os.FileInfo
		info, err = os.Stat(file)
		if err != nil {
			err = fmt.Errorf("Can't check if '%s' is a file or a directory: %s", file, err)
			return
		}
		if info.IsDir() {
			err = l.mergeDir(file)
			if err != nil {
				err = fmt.Errorf("Can't load configuration directory '%s': %s", file, err)
				return
			}
		} else {
			err = l.mergeFile(file)
			if err != nil {
				err = fmt.Errorf("Can't load configuration file '%s': %s", file, err)
				return
			}
		}
	}

	// Return the created configuration:
	config = l.config

	return
}

func (l *Loader) mergeDir(dir string) error {
	// List the files in the directory:
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	files := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.IsDir() {
			name := info.Name()
			if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
				file := filepath.Join(dir, name)
				files = append(files, file)
			}
		}
	}

	// Load the files in alphabetical order:
	sort.Strings(files)
	for _, file := range files {
		err := l.mergeFile(file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) mergeFile(file string) error {
	var err error

	// Read the content of the file:
	glog.Infof("Loading configuration from '%s'", file)
	var content []byte
	content, err = ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// Parse the YAML inside the file:
	var decoded data.Config
	err = yaml.Unmarshal(content, &decoded)
	if err != nil {
		return err
	}

	// Merge the configuration data from the file with the existing configuration:
	if decoded.AWX != nil {
		err = l.mergeAWX(decoded.AWX)
		if err != nil {
			return err
		}
	}
	if decoded.Throttling != nil {
		err = l.mergeThrottling(decoded.Throttling)
		if err != nil {
			return err
		}
	}
	if decoded.Rules != nil {
		err = l.mergeRules(decoded.Rules)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) mergeAWX(decoded *data.AWXConfig) error {
	// Merge the server address and proxy:
	if decoded.Address != "" {
		l.config.awx.address = decoded.Address
	}
	if decoded.Proxy != "" {
		l.config.awx.proxy = decoded.Proxy
	}

	// Merge the credentials:
	if decoded.Credentials != nil {
		err := l.mergeAWXCredentials(decoded.Credentials)
		if err != nil {
			return err
		}
	}
	if decoded.CredentialsRef != nil {
		err := l.mergeAWXCredentialsSecret(decoded.CredentialsRef)
		if err != nil {
			return err
		}
	}

	// Merge the TLS details:
	if decoded.TLS != nil {
		err := l.mergeAWXTLS(decoded.TLS)
		if err != nil {
			return err
		}
	}
	if decoded.TLSRef != nil {
		err := l.mergeAWXTLSSecret(decoded.TLSRef)
		if err != nil {
			return err
		}
	}

	// Merge insecure setting:
	l.config.awx.insecure = decoded.Insecure

	// Merge the project:
	if decoded.Project != "" {
		l.config.awx.project = decoded.Project
	}

	// Merge the jobStatusCheckInterval
	if decoded.JobStatusCheckInterval != "" {
		interval, err := time.ParseDuration(decoded.JobStatusCheckInterval)
		if err != nil {
			return err
		}
		l.config.awx.jobStatusCheckInterval = interval
	}

	return nil
}

func (l *Loader) mergeThrottling(decoded *data.ThrottlingConfig) error {
	if decoded.Interval != "" {
		interval, err := time.ParseDuration(decoded.Interval)
		if err != nil {
			return err
		}
		l.config.throttling.interval = interval
	}
	return nil
}

func (l *Loader) mergeAWXCredentials(credentials *data.AWXCredentialsConfig) error {
	if credentials.Username != "" {
		l.config.awx.user = credentials.Username
	}
	if credentials.Password != "" {
		l.config.awx.password = credentials.Password
	}
	return nil
}

func (l *Loader) mergeAWXCredentialsSecret(reference *core.SecretReference) error {
	secret, err := l.loadSecret(reference)
	if err != nil {
		return err
	}
	if secret.Data != nil {
		var value []byte
		var ok bool
		value, ok = secret.Data[core.BasicAuthUsernameKey]
		if ok {
			l.config.awx.user = string(value)
		}
		value, ok = secret.Data[core.BasicAuthPasswordKey]
		if ok {
			l.config.awx.password = string(value)
		}
	}
	return nil
}

func (l *Loader) mergeAWXTLS(tls *data.TLSConfig) error {
	if tls.CACerts != "" {
		l.config.awx.ca.WriteString(tls.CACerts)
	}
	if tls.CAFile != "" {
		caCerts, err := ioutil.ReadFile(tls.CAFile)
		if err != nil {
			return err
		}
		l.config.awx.ca.Write(caCerts)
	}
	return nil
}

func (l *Loader) mergeAWXTLSSecret(reference *core.SecretReference) error {
	secret, err := l.loadSecret(reference)
	if err != nil {
		return err
	}
	if secret.Data != nil {
		var value []byte
		var ok bool
		value, ok = secret.Data[core.ServiceAccountRootCAKey]
		if ok {
			l.config.awx.ca.Write(value)
		}
	}
	return nil
}

func (l *Loader) loadSecret(reference *core.SecretReference) (secret *core.Secret, err error) {
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
	if l.client == nil {
		err = fmt.Errorf(
			"Can't load secret '%s' from namespace '%s' because there is no connection to the Kubernetes API",
			reference.Name,
			reference.Namespace,
		)
		return
	}

	// Try to retrieve the secret:
	resource := l.client.CoreV1().Secrets(reference.Namespace)
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

func (l *Loader) mergeRules(rules []interface{}) error {
	for _, rule := range rules {
		err := l.mergeRule(rule)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Loader) mergeRule(rawRule interface{}) error {
	// The rule was originally written in YAML inside the configuration file, but in order to
	// deserialize it using the Kubernetes API versioning mechanism we need to convert it back to
	// JSON, as the coded only supports JSON.
	jsonRule, err := json.Marshal(rawRule)
	if err != nil {
		return fmt.Errorf("Can't convert rule to JSON: %s", err)
	}

	// Now we can create an empty instance of the type that we expect and try to convert the JSON
	// produced in the previous step to that type:
	inRule := new(autoheal.HealingRule)
	defaultKind := reflect.TypeOf(*inRule).Name()
	defaultGVK := v1alpha2.SchemeGroupVersion.WithKind(defaultKind)
	outRule, _, err := l.codec.Decode(jsonRule, &defaultGVK, inRule)
	if err != nil {
		return fmt.Errorf("Can't convert rule JSON to type '%s': %s", defaultKind, err)
	}

	// Check that the resulting object is really the type that we expect:
	convertedRule, ok := outRule.(*autoheal.HealingRule)
	if !ok {
		return fmt.Errorf("Converted rule is of type '%T', but expected '%T'", outRule, inRule)
	}

	// Add the rule to the list:
	l.config.rules = append(l.config.rules, convertedRule)

	return nil
}
