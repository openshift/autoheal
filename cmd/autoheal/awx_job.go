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

	alertmanager "github.com/openshift/autoheal/pkg/alertmanager"
	monitoring "github.com/openshift/autoheal/pkg/apis/monitoring/v1alpha1"
	awx "github.com/openshift/autoheal/pkg/awx"
)

func (h *Healer) runAWXJob(rule *monitoring.HealingRule, action *monitoring.AWXJobAction, alert *alertmanager.Alert) error {
	var err error

	// Get the AWX connection details from the configuration:
	awxAddress := h.config.AWX().Address()
	awxProxy := h.config.AWX().Proxy()
	awxUser := h.config.AWX().User()
	awxPassword := h.config.AWX().Password()
	awxCA := h.config.AWX().CA()
	awxInsecure := h.config.AWX().Insecure()

	// Get the name of the AWX project name from the configuration:
	awxProject := h.config.AWX().Project()

	// Get the name of the AWX job template from the action:
	awxTemplate := action.Template

	// Create the connection to the AWX server:
	connection, err := awx.NewConnectionBuilder().
		Url(awxAddress).
		Proxy(awxProxy).
		Username(awxUser).
		Password(awxPassword).
		CACertificates(awxCA).
		Insecure(awxInsecure).
		Build()
	if err != nil {
		return err
	}
	defer connection.Close()

	// Retrieve the job template:
	templatesResource := connection.JobTemplates()
	templatesResponse, err := templatesResource.Get().
		Filter("project__name", awxProject).
		Filter("name", awxTemplate).
		Send()
	if err != nil {
		return err
	}
	if templatesResponse.Count() == 0 {
		return fmt.Errorf(
			"Template '%s' not found in project '%s'",
			awxTemplate,
			awxProject,
		)
	}

	// Launch the jobs:
	glog.Infof(
		"Running AWX job from project '%s' and template '%s' to heal alert '%s'",
		awxProject,
		awxTemplate,
		alert.Name(),
	)
	for _, template := range templatesResponse.Results() {
		err := h.launchAWXJob(connection, template, action, rule)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *Healer) launchAWXJob(
	connection *awx.Connection,
	template *awx.JobTemplate,
	action *monitoring.AWXJobAction,
	rule *monitoring.HealingRule,
) error {
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
	h.incrementAwxActions(action, rule.ObjectMeta.Name)
	return nil
}
