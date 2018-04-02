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

// This file contains the data structures used for requesting authentication tokens.

package data

import (
	"time"
)

// Personal Access Token, user token in OAuth2
type PATPostRequest struct {
	Description string  `json:"description,omitempty"`
	Application *string `json:"application"` // Must be "null" in a PAT request
	Scope       string  `json:"scope,omitempty"`
}

type PATPostResponse struct {
	Token   string    `json:"token,omitempty"`
	Expires time.Time `json:"expires,omitempty"`
}
