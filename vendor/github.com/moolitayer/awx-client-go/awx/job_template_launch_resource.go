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

// This file contains the implementation of the resource that manages launching of jobs from job
// templates.

package awx

import (
	"encoding/json"

	"github.com/moolitayer/awx-client-go/awx/internal/data"
)

type JobTemplateLaunchResource struct {
	Resource
}

func NewJobTemplateLaunchResource(connection *Connection, path string) *JobTemplateLaunchResource {
	resource := new(JobTemplateLaunchResource)
	resource.connection = connection
	resource.path = path
	return resource
}

func (r *JobTemplateLaunchResource) Get() *JobTemplateLaunchGetRequest {
	request := new(JobTemplateLaunchGetRequest)
	request.resource = &r.Resource
	return request
}

func (r *JobTemplateLaunchResource) Post() *JobTemplateLaunchPostRequest {
	request := new(JobTemplateLaunchPostRequest)
	request.resource = &r.Resource
	return request
}

type JobTemplateLaunchGetRequest struct {
	Request
}

func (r *JobTemplateLaunchGetRequest) Send() (response *JobTemplateLaunchGetResponse, err error) {
	output := new(data.JobTemplateLaunchGetResponse)
	err = r.get(output)
	if err != nil {
		return
	}
	response = new(JobTemplateLaunchGetResponse)
	if output.JobTemplateData != nil {
		response.jobTemplateData = new(JobTemplate)
		response.jobTemplateData.id = output.JobTemplateData.Id
		response.jobTemplateData.name = output.JobTemplateData.Name
	}
	return
}

type JobTemplateLaunchGetResponse struct {
	jobTemplateData *JobTemplate
}

func (r *JobTemplateLaunchGetResponse) JobTemplateData() *JobTemplate {
	return r.jobTemplateData
}

type JobTemplateLaunchPostRequest struct {
	Request

	extraVars map[string]interface{}
	limit     string
}

// ExtraVars set a map or external variables sent to the AWX job.
func (r *JobTemplateLaunchPostRequest) ExtraVars(value map[string]interface{}) *JobTemplateLaunchPostRequest {
	r.extraVars = value
	return r
}

// ExtraVar adds a single external variable to extraVars map.
func (r *JobTemplateLaunchPostRequest) ExtraVar(name string, value interface{}) *JobTemplateLaunchPostRequest {
	if r.extraVars == nil {
		r.extraVars = make(map[string]interface{})
	}
	r.extraVars[name] = value
	return r
}

// Limit allows limiting template execution to specific hosts.
func (r *JobTemplateLaunchPostRequest) Limit(value string) *JobTemplateLaunchPostRequest {
	r.limit = value
	return r
}

func (r *JobTemplateLaunchPostRequest) Send() (response *JobTemplateLaunchPostResponse, err error) {
	// Generate the input data:
	input := new(data.JobTemplateLaunchPostRequest)

	if r.extraVars != nil {
		// convert extravars json to string
		var bytes []byte
		bytes, err = json.Marshal(r.extraVars)
		if err != nil {
			return
		}
		input.ExtraVars = string(bytes)
	}

	input.Limit = r.limit

	// Send the request:
	output := new(data.JobTemplateLaunchPostResponse)
	err = r.post(input, output)
	if err != nil {
		return
	}

	// Analyze the output data:
	response = new(JobTemplateLaunchPostResponse)
	response.Job = output.Job

	return
}

type JobTemplateLaunchPostResponse struct {
	Job int
}
