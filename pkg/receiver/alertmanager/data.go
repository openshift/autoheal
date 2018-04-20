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

package alertmanager

import (
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"sort"
	"time"
)

// AlertStatus represents the status of a alert.
//
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

// Data represents each message sent by the alert manager to a receiver.
//
type Message struct {
	Receiver          string            `json:"receiver,omitempty"`
	Status            AlertStatus       `json:"status,omitempty"`
	Alerts            []*Alert          `json:"alerts,omitempty"`
	GroupLabels       map[string]string `json:"groupLabels,omitempty"`
	CommonLabels      map[string]string `json:"commonLabels,omitempty"`
	CommonAnnotations map[string]string `json:"commonAnnotations,omitempty"`
	ExternalURL       string            `json:"exterlalURL,omitempty"`
}

// Alert represents each of the alerts sent by the alert manager to a receiver.
//
type Alert struct {
	Status       AlertStatus       `json:"status,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	StartsAt     time.Time         `json:"startsAt,omitempty"`
	EndsAt       time.Time         `json:"endsAt,omitempty"`
	GeneratorURL time.Time         `json:"generatorURL,omitempty"`
}

// Name returns the name of the alert.
//
func (a *Alert) Name() string {
	return a.Labels["alertname"]
}

// Namespace returns the namespace of the alert.
//
func (a *Alert) Namespace() string {
	namespace := a.Labels["namespace"]
	if namespace == "" {
		namespace = a.Annotations["namespace"]
	}
	if namespace == "" {
		namespace = "default"
	}
	return namespace
}

// Hash calculates the hash for the alert.
//
func (a *Alert) Hash() string {
	dst := fnv.New32a()
	hashMap(a.Labels, dst)
	io.WriteString(dst, "\n")
	hashMap(a.Annotations, dst)
	sum := dst.Sum32()
	return fmt.Sprintf("%d", sum)
}

// hashMap writes the keys and values of a map to a hash, making sure that they are in order to
// that the result will allways be the same regardless of the internal ordering of the map.
//
func hashMap(src map[string]string, dst hash.Hash) {
	keys := make([]string, len(src))
	i := 0
	for key := range src {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := src[key]
		io.WriteString(dst, key)
		io.WriteString(dst, "=")
		io.WriteString(dst, value)
		io.WriteString(dst, "\n")
	}
}
