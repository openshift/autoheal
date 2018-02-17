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

// This file contains the functions to calculate the hashes fo alerts.

package main

import (
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"sort"

	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
)

// hashAlert calculates the hash for the given alert.
//
func hashAlert(alert *monitoring.Alert) string {
	dst := fnv.New32a()
	hashMap(alert.Status.Labels, dst)
	io.WriteString(dst, "\n")
	hashMap(alert.Status.Annotations, dst)
	sum := dst.Sum32()
	return fmt.Sprintf("%d", sum)
}

// hashMap writes the keys and values of a map to a hash, making sure that they are in order to
// that the result will allways be the same regardless of the internal ordering of the map.
//
func hashMap(src map[string]string, dst hash.Hash) {
	keys := make([]string, len(src))
	i := 0
	for key, _ := range src {
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
