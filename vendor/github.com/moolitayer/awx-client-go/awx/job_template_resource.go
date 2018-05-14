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

// This file contains the implementation of the resource that manages a specific job
// template.

package awx

import (
	"github.com/moolitayer/awx-client-go/awx/internal/data"
)

type JobTemplateResource struct {
	Resource
}

func NewJobTemplateResource(connection *Connection, path string) *JobTemplateResource {
	resource := new(JobTemplateResource)
	resource.connection = connection
	resource.path = path
	return resource
}

func (r *JobTemplateResource) Get() *JobTemplateGetRequest {
	request := new(JobTemplateGetRequest)
	request.resource = &r.Resource
	return request
}

func (r *JobTemplateResource) Launch() *JobTemplateLaunchResource {
	return NewJobTemplateLaunchResource(r.connection, r.path+"/launch")
}

type JobTemplateGetRequest struct {
	Request
}

func (r *JobTemplateGetRequest) Send() (response *JobTemplateGetResponse, err error) {
	output := new(data.JobTemplateGetResponse)
	err = r.get(output)
	if err != nil {
		return
	}
	response = new(JobTemplateGetResponse)
	response.result = new(JobTemplate)
	response.result.id = output.Id
	response.result.name = output.Name
	response.result.askLimitOnLaunch = output.AskLimitOnLaunch
	response.result.askVarsOnLaunch = output.AskVarsOnLaunch

	return
}

type JobTemplateGetResponse struct {
	result *JobTemplate
}

func (r *JobTemplateGetResponse) Result() *JobTemplate {
	return r.result
}
