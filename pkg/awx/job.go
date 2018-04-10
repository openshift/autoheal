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

// This file contains the implementation of the job template type.

package awx

type JobStatus string

const (
	JobStatusNew       JobStatus = "new"
	JobStatusPending   JobStatus = "pending"
	JobStatusWaiting   JobStatus = "waiting"
	JobStatusRunning   JobStatus = "running"
	JobStatusSuccesful JobStatus = "successful"
	JobStatusFailed    JobStatus = "failed"
	JobStatusError     JobStatus = "error"
	JobStatusCancelled JobStatus = "cancelled"
)

type Job struct {
	id     int
	status JobStatus
}

func (j *Job) Id() int {
	return j.id
}

func (j *Job) Status() JobStatus {
	return j.status
}

func (j *Job) IsFinished() bool {
	switch j.status {
	case
		JobStatusSuccesful,
		JobStatusFailed,
		JobStatusError,
		JobStatusCancelled:
		return true
	}
	return false
}
