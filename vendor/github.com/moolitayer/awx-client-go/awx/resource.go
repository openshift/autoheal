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

// This file contains the basic implementation shared by all the resources.

package awx

import (
	"net/url"
)

type Resource struct {
	connection *Connection
	path       string
}

func (r *Resource) get(query url.Values, output interface{}) error {
	return r.connection.authenticatedGet(r.path, query, output)
}

func (r *Resource) post(query url.Values, input interface{}, output interface{}) error {
	return r.connection.authenticatedPost(r.path, query, input, output)
}

func (r *Resource) String() string {
	return r.path
}
