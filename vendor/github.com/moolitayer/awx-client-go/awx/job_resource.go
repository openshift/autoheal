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
	"github.com/moolitayer/awx-client-go/awx/internal/data"
)

type JobResource struct {
	Resource
}

func NewJobResource(connection *Connection, path string) *JobResource {
	resource := new(JobResource)
	resource.connection = connection
	resource.path = path
	return resource
}

func (r *JobResource) Get() *JobGetRequest {
	request := new(JobGetRequest)
	request.resource = &r.Resource
	return request
}

type JobGetRequest struct {
	Request
}

func (r *JobGetRequest) Send() (response *JobGetResponse, err error) {
	output := new(data.JobGetResponse)
	err = r.get(output)
	if err != nil {
		return nil, err
	}
	response = new(JobGetResponse)
	if output != nil {
		response.job = new(Job)
		response.job.id = output.Id
		response.job.status = (JobStatus)(output.Status)
	}
	return
}

type JobGetResponse struct {
	job *Job
}

func (r *JobGetResponse) Job() *Job {
	return r.job
}
