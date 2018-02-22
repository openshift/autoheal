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
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
)

func (h *Healer) runAnsiblePlaybook(rule *monitoring.HealingRule, action *monitoring.AnsiblePlaybookAction, alert *alertmanager.Alert) error {
	glog.Infof(
		"Running Ansible playbook from healing rule '%s' and alert '%s'",
		rule.ObjectMeta.Name,
		alert.Labels["alertname"],
	)

	// The configuration map and the job will be in the same namespace and will have the same name
	// than the alert:
	namespace := alert.Labels["namespace"]
	name := alert.Labels["alertname"]

	// Populate the configuration map:
	config := &core.ConfigMap{
		ObjectMeta: meta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string]string{
			"playbook.yml": action.Playbook,
			"inventory":    action.Inventory,
		},
	}

	// Create the configuration map:
	configsResource := h.k8sClient.Core().ConfigMaps(namespace)
	_, err := configsResource.Create(config)
	if errors.IsAlreadyExists(err) {
		glog.Infof(
			"Configuration map '%s' already exists",
			config.ObjectMeta.Name,
		)
	} else if err != nil {
		return err
	} else {
		glog.Infof(
			"Created configuration map '%s'",
			config.ObjectMeta.Name,
		)
	}

	// Determine the user that will be used to run Ansible. This is needed because the UID must
	// match the permissions set inside the image, otherwise Ansible will not be able to write to
	// the home directory.
	uid := int64(1000000000)

	// Build the Ansible command line:
	command := []string{
		"/usr/bin/ansible-playbook",
		"--inventory=/config/inventory",
	}
	if action.ExtraVars != "" {
		command = append(command, "--extra-vars="+action.ExtraVars)
	}
	command = append(command, "/config/playbook.yml")

	// Populate the job:
	job := &batch.Job{
		ObjectMeta: meta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: batch.JobSpec{
			Template: core.PodTemplateSpec{
				Spec: core.PodSpec{
					Volumes: []core.Volume{
						core.Volume{
							Name: "config",
							VolumeSource: core.VolumeSource{
								ConfigMap: &core.ConfigMapVolumeSource{
									LocalObjectReference: core.LocalObjectReference{
										Name: config.ObjectMeta.Name,
									},
								},
							},
						},
					},
					Containers: []core.Container{
						core.Container{
							Name:  "ansible-runner",
							Image: "openshift-monitoring/ansible-runner:0.0.0",
							SecurityContext: &core.SecurityContext{
								RunAsUser: &uid,
							},
							Command: command,
							VolumeMounts: []core.VolumeMount{
								core.VolumeMount{
									Name:      "config",
									MountPath: "/config",
								},
							},
						},
					},
					RestartPolicy: core.RestartPolicyNever,
				},
			},
		},
	}

	// Create the job:
	jobsResource := h.k8sClient.Batch().Jobs(namespace)
	_, err = jobsResource.Create(job)
	if errors.IsAlreadyExists(err) {
		glog.Infof(
			"Job '%s' already exists",
			job.ObjectMeta.Name,
		)
	} else if err != nil {
		return err
	} else {
		glog.Infof(
			"Created job '%s'",
			job.ObjectMeta.Name,
		)
	}

	return nil
}
