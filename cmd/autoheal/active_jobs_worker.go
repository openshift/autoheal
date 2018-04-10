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

package main

import (
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/runtime"
)

func (h *Healer) runActiveJobsWorker() {
	glog.Infof("Going over active jobs queue.")

	finishedJobs := make([]int, 0)

	h.activeJobs.Range(func(_, value interface{}) bool {
		id := value.(int)

		finished, err := h.checkAWXJobStatus(id)
		if err != nil {
			runtime.HandleError(err)
		}

		if finished {
			finishedJobs = append(finishedJobs, id)
		}
		return true
	})

	// remove finished jobs from the queue
	for _, job := range finishedJobs {
		glog.Infof(
			"Removing finished job `%v` from queue ",
			job,
		)
		h.activeJobs.Delete(job)
	}
}
