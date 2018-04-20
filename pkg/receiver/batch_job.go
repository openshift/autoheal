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

package receiver

import (
	"fmt"

	"github.com/golang/glog"
	batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/receiver/alertmanager"
)

func (h *Healer) runBatchJob(rule *autoheal.HealingRule, job *batch.Job, alert *alertmanager.Alert) error {
	glog.Infof(
		"Running batch job '%s' to heal alert '%s'",
		job.ObjectMeta.Name,
		alert.Labels["alertname"],
	)

	// The name of the job is mandatory:
	name := job.ObjectMeta.Name
	if name == "" {
		return fmt.Errorf(
			"Can't create job for rule '%s', the name hasn't been specified",
			rule.ObjectMeta.Name,
		)
	}

	// The namespace of the job is optional, the default is the namespace of the rule:
	namespace := job.ObjectMeta.Namespace
	if namespace == "" {
		namespace = rule.ObjectMeta.Namespace
	}

	// Get the resource that manages the collection of batch jobs:
	resource := h.k8sClient.Batch().Jobs(namespace)

	// Try to create the job:
	job = job.DeepCopy()
	job.ObjectMeta.Name = name
	job.ObjectMeta.Namespace = namespace
	_, err := resource.Create(job)
	if errors.IsAlreadyExists(err) {
		glog.Warningf(
			"Batch job '%s' already exists, will do nothing to heal alert '%s'",
			job.ObjectMeta.Name,
			alert.Labels["alertname"],
		)
	} else if err != nil {
		return err
	} else {
		glog.Infof(
			"Batch job '%s' to heal alert '%s' has been created",
			job.ObjectMeta.Name,
			alert.Labels["alertname"],
		)
	}

	return nil
}
