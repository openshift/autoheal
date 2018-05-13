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

// This file contains the implementation of the resource that manages the collection of
// job templates.

package awx

import (
	"fmt"

	"github.com/moolitayer/awx-client-go/awx/internal/data"
)

type JobTemplatesResource struct {
	Resource
}

func NewJobTemplatesResource(connection *Connection, path string) *JobTemplatesResource {
	resource := new(JobTemplatesResource)
	resource.connection = connection
	resource.path = path
	return resource
}

func (r *JobTemplatesResource) Get() *JobTemplatesGetRequest {
	request := new(JobTemplatesGetRequest)
	request.resource = &r.Resource
	return request
}

func (r *JobTemplatesResource) Id(id int) *JobTemplateResource {
	return NewJobTemplateResource(r.connection, fmt.Sprintf("%s/%d", r.path, id))
}

type JobTemplatesGetRequest struct {
	Request
}

func (r *JobTemplatesGetRequest) Filter(name string, value interface{}) *JobTemplatesGetRequest {
	r.addFilter(name, value)
	return r
}

func (r *JobTemplatesGetRequest) Send() (response *JobTemplatesGetResponse, err error) {
	output := new(data.JobTemplatesGetResponse)
	err = r.get(output)
	if err != nil {
		return
	}
	response = new(JobTemplatesGetResponse)
	response.count = output.Count
	response.previous = output.Previous
	response.next = output.Next
	response.results = make([]*JobTemplate, len(output.Results))
	for i := 0; i < len(output.Results); i++ {
		response.results[i] = new(JobTemplate)
		response.results[i].id = output.Results[i].Id
		response.results[i].name = output.Results[i].Name
		response.results[i].askLimitOnLaunch = output.Results[i].AskLimitOnLaunch
		response.results[i].askVarsOnLaunch = output.Results[i].AskVarsOnLaunch
	}
	return
}

type JobTemplatesGetResponse struct {
	ListGetResponse

	results []*JobTemplate
}

func (r *JobTemplatesGetResponse) Results() []*JobTemplate {
	return r.results
}
