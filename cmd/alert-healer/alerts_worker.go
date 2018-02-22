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
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"text/template"

	"github.com/golang/glog"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"

	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	awx "github.com/jhernand/openshift-monitoring/pkg/awx"
)

func (h *Healer) runAlertsWorker() {
	for h.pickAlert() {
		// Nothing.
	}
}

func (h *Healer) pickAlert() bool {
	// Get the next item and end the work loop if asked to stop:
	item, stop := h.alertsQueue.Get()
	if stop {
		return false
	}

	// Process the item and make sure to always tell the queue that we are done with this item:
	err := func(item interface{}) error {
		h.alertsQueue.Done(item)

		// Check that the item we got from the queue is really an alert, and discard it otherwise:
		alert, ok := item.(*alertmanager.Alert)
		if !ok {
			h.alertsQueue.Forget(item)
		}

		// Process and then forget the alert:
		err := h.processAlert(alert)
		if err != nil {
			return err
		}
		h.alertsQueue.Forget(alert)

		return nil
	}(item)
	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (h *Healer) processAlert(alert *alertmanager.Alert) error {
	switch alert.Status {
	case alertmanager.AlertStatusFiring:
		return h.startHealing(alert)
	case alertmanager.AlertStatusResolved:
		return h.cancelHealing(alert)
	default:
		glog.Warningf(
			"Unknnown status '%s' reported by alert manager, will ignore it",
			alert.Status,
		)
		return nil
	}
}

// startHealing starts the healing process for the given alert.
//
func (h *Healer) startHealing(alert *alertmanager.Alert) error {
	// Find the rules that are activated for the alert:
	activated := make([]*monitoring.HealingRule, 0)
	h.rulesCache.Range(func(_, value interface{}) bool {
		rule := value.(*monitoring.HealingRule)
		if h.checkConditions(rule, alert) {
			glog.Infof(
				"Healing rule '%s' matches alert '%s'",
				rule.ObjectMeta.Name,
				alert.Labels["alertname"],
			)
			activated = append(activated, rule)
		}
		return true
	})
	if len(activated) == 0 {
		glog.Infof("No healing rule matches alert '%s'", alert.Labels["alertname"])
		return nil
	}

	// Execute the actions of the activated rules:
	for _, rule := range activated {
		err := h.runActions(rule, alert)
		if err != nil {
			return err
		}
	}

	return nil
}

// cancelHealing cancels the healing process for the given alert.
//
func (h *Healer) cancelHealing(alert *alertmanager.Alert) error {
	return nil
}

func (h *Healer) checkConditions(rule *monitoring.HealingRule, alert *alertmanager.Alert) bool {
	glog.Infof(
		"Checking conditions of rule '%s' for alert '%s'",
		rule.ObjectMeta.Name,
		alert.Labels["alertname"],
	)
	if rule.Spec.Conditions != nil && len(rule.Spec.Conditions) > 0 {
		for i := 0; i < len(rule.Spec.Conditions); i++ {
			if !h.checkCondition(&rule.Spec.Conditions[i], alert) {
				return false
			}
		}
	}
	return true
}

func (h *Healer) checkCondition(condition *monitoring.HealingCondition, alert *alertmanager.Alert) bool {
	matched, err := regexp.MatchString(condition.Alert, alert.Labels["alertname"])
	if err != nil {
		glog.Errorf(
			"Error while checking if alert name '%s' matches pattern '%s': %s",
			alert.Labels["alertname"],
			condition.Alert,
			err.Error(),
		)
		matched = false
	}
	return matched
}

func (h *Healer) runActions(rule *monitoring.HealingRule, alert *alertmanager.Alert) error {
	if rule.Spec.Actions != nil && len(rule.Spec.Actions) > 0 {
		glog.Infof(
			"Running actions of healing rule '%s' for alert '%s'",
			rule.ObjectMeta.Name,
			alert.Labels["alertname"],
		)
		for i := 0; i < len(rule.Spec.Actions); i++ {
			err := h.runAction(rule, &rule.Spec.Actions[i], alert)
			if err != nil {
				return err
			}
		}
	} else {
		glog.Warningf(
			"Healing rule '%s' has no actions, will have no effect on alert '%s'",
			rule.ObjectMeta.Name,
			alert.Labels["alertname"],
		)
	}

	return nil
}

func (h *Healer) runAction(rule *monitoring.HealingRule, action *monitoring.HealingAction, alert *alertmanager.Alert) error {
	// Convert the action to a JSON document, so that we can easily process it as a temlate:
	blob, err := json.Marshal(action)
	if err != nil {
		return err
	}
	glog.Infof(
		"Action before processing template:\n%s",
		h.indent(blob),
	)

	// Generate the template, adding some convenience variables:
	buffer := new(bytes.Buffer)
	buffer.WriteString("{{ $alert := . }}\n")
	buffer.WriteString("{{ $labels := .Labels }}\n")
	buffer.WriteString("{{ $annotations := .Annotations }}\n")
	buffer.Write(blob)
	text := buffer.String()
	glog.Infof(
		"Generated template:\n%s",
		text,
	)

	// Parse and run the template:
	tmpl, err := template.New("action").Parse(text)
	if err != nil {
		return err
	}
	buffer.Reset()
	err = tmpl.Execute(buffer, alert)
	if err != nil {
		return err
	}
	blob = buffer.Bytes()
	glog.Infof(
		"Action after processing template:\n%s",
		h.indent(blob),
	)

	// Convert the processed JSON document back to an action:
	action = new(monitoring.HealingAction)
	err = json.Unmarshal(blob, action)
	if err != nil {
		return err
	}

	// Check the type of action and run it:
	if action.AWXJob != nil {
		return h.runAWXJob(rule, action.AWXJob, alert)
	} else if action.BatchJob != nil {
		return h.runBatchJob(rule, action.BatchJob, alert)
	} else if action.AnsiblePlaybook != nil {
		return h.runAnsiblePlaybook(rule, action.AnsiblePlaybook, alert)
	} else {
		glog.Warningf(
			"There are no action details, rule '%s' will have no effect on alert '%s'",
			rule.ObjectMeta.Name,
			alert.Labels["alertname"],
		)
	}

	return nil
}

func (h *Healer) runAWXJob(rule *monitoring.HealingRule, action *monitoring.AWXJobAction, alert *alertmanager.Alert) error {
	glog.Infof(
		"Running AWX job from project '%s' and template '%s' to heal alert '%s'",
		action.Project,
		action.Template,
		alert.Labels["alertname"],
	)

	// Load the AWX credentials:
	secret := action.SecretRef
	if secret == nil {
		return fmt.Errorf("The secret containing the AWX credentials hasn't been specified")
	}
	username, password, err := h.loadAWXSecret(rule, secret)
	if err != nil {
		return err
	}

	// Create the connection to the AWX server:
	connection, err := awx.NewConnectionBuilder().
		Url(action.Address).
		Proxy(action.Proxy).
		Username(username).
		Password(password).
		Insecure(true).
		Build()
	if err != nil {
		return err
	}
	defer connection.Close()

	// Retrieve the job template:
	templatesResource := connection.JobTemplates()
	templatesResponse, err := templatesResource.Get().
		Filter("project__name", action.Project).
		Filter("name", action.Template).
		Send()
	if err != nil {
		return err
	}
	if templatesResponse.Count() == 0 {
		return err
	}

	// Launch the jobs:
	for _, template := range templatesResponse.Results() {
		err := h.launchAWXJob(connection, template, action)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *Healer) launchAWXJob(connection *awx.Connection, template *awx.JobTemplate, action *monitoring.AWXJobAction) error {
	templateId := template.Id()
	templateName := template.Name()
	launchResource := connection.JobTemplates().Id(templateId).Launch()
	_, err := launchResource.Post().
		ExtraVars(action.ExtraVars).
		Send()
	if err != nil {
		return err
	}
	glog.Infof(
		"Request to launch AWX job from template '%s' has been sent",
		templateName,
	)
	return nil
}

func (h *Healer) loadAWXSecret(rule *monitoring.HealingRule, reference *core.SecretReference) (username, password string, err error) {
	var data []byte
	var ok bool

	// The name of the secret is mandatory:
	name := reference.Name
	if name == "" {
		err = fmt.Errorf(
			"Can't load AWX secret for rule '%s', the name hasn't been specified",
			rule.ObjectMeta.Name,
		)
		return
	}

	// The namespace of the secret is optional, the default is the namespace of the rule:
	namespace := reference.Namespace
	if namespace == "" {
		namespace = rule.ObjectMeta.Namespace
	}

	// Retrieve the secret:
	resource := h.k8sClient.CoreV1().Secrets(namespace)
	secret, err := resource.Get(name, meta.GetOptions{})
	if err != nil {
		err = fmt.Errorf(
			"Can't load secret '%s' from namespace '%s': %s",
			name,
			namespace,
			err.Error(),
		)
		return
	}

	// Extract the user name:
	data, ok = secret.Data["username"]
	if !ok {
		err = fmt.Errorf(
			"Secret '%s' from namespace '%s' doesn't contain the 'username' entry",
			name,
			namespace,
		)
		return
	}
	username = string(data)

	// Extract the password:
	data, ok = secret.Data["password"]
	if !ok {
		err = fmt.Errorf(
			"Secret '%s' from namespace '%s' doesn't contain the 'password' entry",
			name,
			namespace,
		)
		return
	}
	password = string(data)

	return
}

func (h *Healer) runBatchJob(rule *monitoring.HealingRule, job *batch.Job, alert *alertmanager.Alert) error {
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
