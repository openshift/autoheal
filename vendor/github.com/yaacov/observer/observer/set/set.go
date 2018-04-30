// Copyright 2018 Yaacov Zamir <kobi.zamir@gmail.com>
// and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package set for unique collection of strings.
package set

import (
	"fmt"
)

// Set objects are collections of strings. A value in the Set may only occur once,
// it is unique in the Set's collection.
type Set struct {
	set map[string]struct{}
}

// Add appends a new element with the given value to the Set object.
// It returns an error if the value already in set.
func (s *Set) Add(v string) error {
	// Init the set map if set is empty.
	if s.set == nil {
		s.set = make(map[string]struct{})
	}

	// Check for value exist.
	if _, ok := s.set[v]; ok {
		return fmt.Errorf("Value already in set.")
	}

	s.set[v] = struct{}{}
	return nil
}

// Clear removes all elements from the Set object.
func (s *Set) Clear() {
	s.set = nil
}

// Values returns a new list object that contains the values for each element
// in the Set object.
func (s Set) Values() (keys []string) {
	keys = make([]string, 0, len(s.set))
	for k := range s.set {
		keys = append(keys, k)
	}

	return
}

// Has returns a boolean asserting whether an element is present with the
// given value in the Set object or not.
func (s Set) Has(v string) (ok bool) {
	_, ok = s.set[v]

	return
}
