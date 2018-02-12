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
	"net/url"
)

type Request struct {
	resource *Resource
	query    url.Values
}

func (r *Request) addFilter(name string, value interface{}) {
	if r.query == nil {
		r.query = make(url.Values)
	}
	r.query.Add(name, fmt.Sprintf("%s", value))
}

func (r *Request) get(output interface{}) error {
	return r.resource.get(r.query, output)
}

func (r *Request) post(input interface{}, output interface{}) error {
	return r.resource.post(r.query, input, output)
}
