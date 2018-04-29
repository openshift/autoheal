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

// This file contains the definitions of the unversioned types used internally by the auto-heal
// service.

package autoheal

import (
	"reflect"
	"testing"
)

func TestJsonDocDeepCopy(t *testing.T) {
	in := JsonDoc{
		"name": map[string]interface{}{
			"first": "John",
			"last":  "Snow",
		},
		"title": "King in the North",
	}

	out := in.DeepCopy()

	if !reflect.DeepEqual(out, in) {
		t.Fatalf("\nExpected: %v\nActual:   %v", in, out)
	}

}
