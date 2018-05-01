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

	"github.com/openshift/autoheal/pkg/internal/data"
)

// ThrottlingConfig is a read only view of the section of the configuration that describes how to
// throttle the execution of healing rules.
//
type ThrottlingConfig struct {
	interval time.Duration
}

// Interval returns the throttling interval for the execution of the actions defined in the healing
// rules.
//
func (t *ThrottlingConfig) Interval() time.Duration {
	return t.interval
}

func (t *ThrottlingConfig) merge(decoded *data.ThrottlingConfig) error {
	if decoded.Interval != "" {
		interval, err := time.ParseDuration(decoded.Interval)
		if err != nil {
			return err
		}
		t.interval = interval
	}
	return nil
}
