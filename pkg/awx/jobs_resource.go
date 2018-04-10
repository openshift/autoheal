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

	"github.com/openshift/autoheal/pkg/awx/internal/data"
)

type JobsResource struct {
	Resource
}

func NewJobsResource(connection *Connection, path string) *JobsResource {
	resource := new(JobsResource)
	resource.connection = connection
	resource.path = path
	return resource
}

func (r *JobsResource) Get() *JobsGetRequest {
	request := new(JobsGetRequest)
	request.resource = &r.Resource
	return request
}

func (r *JobsResource) Id(id int) *JobResource {
	return NewJobResource(r.connection, fmt.Sprintf("%s/%d", r.path, id))
}

type JobsGetRequest struct {
	Request
}

func (r *JobsGetRequest) Filter(name string, value interface{}) *JobsGetRequest {
	r.addFilter(name, value)
	return r
}

func (r *JobsGetRequest) Send() (response *JobsGetResponse, err error) {
	output := new(data.JobsGetResponse)
	err = r.get(output)
	if err != nil {
		return
	}
	response = new(JobsGetResponse)
	response.count = output.Count
	response.previous = output.Previous
	response.next = output.Next
	response.results = make([]*Job, len(output.Results))
	for i, result := range response.results {
		result = new(Job)
		result.id = output.Results[i].Id
		result.status = (JobStatus)(output.Results[i].Status)
	}
	return
}

type JobsGetResponse struct {
	ListGetResponse

	results []*Job
}

func (r *JobsGetResponse) Results() []*Job {
	return r.results
}
