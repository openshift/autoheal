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

// This file contains the the implementation of the capabilities that are common to any kind of
// request.

package awx

import (
	"fmt"
)

type ListRequest struct {
	filters map[string]string
}

func (r *ListRequest) addFilter(name string, value interface{}) {
	if r.filters == nil {
		r.filters = make(map[string]string)
	}
	r.filters[name] = fmt.Sprintf("%s", value)
}
