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

// This file contains the implementation of the resource that manages a specific project.

package awx

import (
	"github.com/moolitayer/awx-client-go/awx/internal/data"
)

type ProjectResource struct {
	Resource
}

func NewProjectResource(connection *Connection, path string) *ProjectResource {
	resource := new(ProjectResource)
	resource.connection = connection
	resource.path = path
	return resource
}

func (r *ProjectResource) Get() *ProjectGetRequest {
	request := new(ProjectGetRequest)
	request.resource = &r.Resource
	return request
}

type ProjectGetRequest struct {
	Request
}

func (r *ProjectGetRequest) Send() (response *ProjectGetResponse, err error) {
	output := new(data.ProjectGetResponse)
	err = r.get(output)
	if err != nil {
		return
	}
	response = new(ProjectGetResponse)
	response.result = new(Project)
	response.result.id = output.Id
	response.result.name = output.Name
	response.result.scmType = output.SCMType
	response.result.scmURL = output.SCMURL
	response.result.scmBranch = output.SCMBranch
	return
}

type ProjectGetResponse struct {
	result *Project
}

func (r *ProjectGetResponse) Result() *Project {
	return r.result
}
