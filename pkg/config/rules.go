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
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/apis/autoheal/v1alpha2"
)

// RulesConfig is a read only view of the section of the configuration that describes
// the healing rules.
//
type RulesConfig struct {
	rules []*autoheal.HealingRule

	// The codec that will be used to convert the rules specified in the configuration file into the
	// types used internally.
	codec runtime.Codec

	// rules array mutex
	rulesMutex *sync.Mutex
}

func (r *RulesConfig) merge(rules []interface{}) error {
	for _, rule := range rules {
		err := r.mergeRule(rule)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RulesConfig) mergeRule(rawRule interface{}) error {
	// Init the rules mutex
	r.init()

	// Lock this function
	r.rulesMutex.Lock()
	defer r.rulesMutex.Unlock()

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
	outRule, _, err := r.codec.Decode(jsonRule, &defaultGVK, inRule)
	if err != nil {
		return fmt.Errorf("Can't convert rule JSON to type '%s': %s", defaultKind, err)
	}

	// Check that the resulting object is really the type that we expect:
	convertedRule, ok := outRule.(*autoheal.HealingRule)
	if !ok {
		return fmt.Errorf("Converted rule is of type '%T', but expected '%T'", outRule, inRule)
	}

	// Add the rule to the list:
	r.rules = append(r.rules, convertedRule)

	return nil
}

// clear the healing rules array
func (r *RulesConfig) clear() {
	// Init the rules mutex
	r.init()

	// Lock this function
	r.rulesMutex.Lock()
	defer r.rulesMutex.Unlock()

	r.rules = nil
}

// init the rules mutex
func (r *RulesConfig) init() {
	if r.rulesMutex == nil {
		r.rulesMutex = &sync.Mutex{}
	}
}
