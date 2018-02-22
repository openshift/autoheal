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
	"fmt"

	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	alertmanager "github.com/jhernand/openshift-monitoring/pkg/alertmanager"
	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	awx "github.com/jhernand/openshift-monitoring/pkg/awx"
)

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
