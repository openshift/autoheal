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

package awxrunner

import (
	"github.com/golang/glog"
	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/metrics"
	"k8s.io/apimachinery/pkg/util/runtime"
)

func (r *Runner) runActiveJobsWorker() {
	glog.Infof("Going over active jobs queue.")

	finishedJobs := make([]int, 0)

	r.activeJobs.Range(func(key interface{}, value interface{}) bool {
		id := key.(int)
		rule := value.(*autoheal.HealingRule)
		finished, err := r.checkAWXJobStatus(id)
		if err != nil {
			runtime.HandleError(err)
		}

		if finished {
			finishedJobs = append(finishedJobs, id)
			metrics.ActionCompleted(
				"AWXJob",
				rule.AWXJob.Template,
				rule.ObjectMeta.Name,
			)
		}
		return true
	})

	// remove finished jobs from the queue
	for _, job := range finishedJobs {
		glog.Infof(
			"Removing finished job `%v` from queue ",
			job,
		)
		r.activeJobs.Delete(job)
	}
}
