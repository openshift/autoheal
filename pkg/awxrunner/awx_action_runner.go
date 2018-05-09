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
	"fmt"

	"github.com/golang/glog"
	"golang.org/x/sync/syncmap"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/moolitayer/awx-client-go/awx"
	"github.com/openshift/autoheal/pkg/alertmanager"
	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/config"
	"github.com/openshift/autoheal/pkg/metrics"
)

type Builder struct {
	config *config.AWXConfig

	stopCh <-chan struct{}
}

type Runner struct {
	config *config.AWXConfig

	activeJobs *syncmap.Map
}

func NewBuilder() *Builder {
	return new(Builder)
}

func (b *Builder) Config(config *config.AWXConfig) *Builder {
	b.config = config
	return b
}

func (b *Builder) StopCh(stopCh <-chan struct{}) *Builder {
	b.stopCh = stopCh
	return b
}

func (b *Builder) Build() (*Runner, error) {
	runner := &Runner{
		config:     b.config,
		activeJobs: new(syncmap.Map),
	}
	go wait.Until(runner.runActiveJobsWorker, runner.config.JobStatusCheckInterval(), b.stopCh)
	return runner, nil
}

func (r *Runner) RunAction(rule *autoheal.HealingRule, action interface{}, alert *alertmanager.Alert) error {
	var err error
	awxAction := action.(*autoheal.AWXJobAction)
	// Get the AWX connection details from the configuration:
	awxAddress := r.config.Address()
	awxProxy := r.config.Proxy()
	awxUser := r.config.User()
	awxPassword := r.config.Password()
	awxCA := r.config.CA()
	awxInsecure := r.config.Insecure()

	// Get the name of the AWX project name from the configuration:
	awxProject := r.config.Project()

	// Get the name of the AWX job template from the action:
	awxTemplate := awxAction.Template

	// Create the connection to the AWX server:
	connection, err := awx.NewConnectionBuilder().
		URL(awxAddress).
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
		err := r.launchAWXJob(connection, template, awxAction, rule, alert)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) launchAWXJob(
	connection *awx.Connection,
	template *awx.JobTemplate,
	action *autoheal.AWXJobAction,
	rule *autoheal.HealingRule,
	alert *alertmanager.Alert,
) error {
	templateId := template.Id()
	templateName := template.Name()
	launchResource := connection.JobTemplates().Id(templateId).Launch()
	response, err := launchResource.Post().
		ExtraVars(action.ExtraVars).
		ExtraVar("alert", alert).
		Limit(action.Limit).
		Send()
	if err != nil {
		return err
	}
	glog.Infof(
		"Request to launch AWX job from template '%s' has been sent, job identifier is '%v'",
		templateName,
		response.Job,
	)
	metrics.ActionStarted(
		"AWXJob",
		templateName,
		rule.ObjectMeta.Name,
	)

	// Add the job to active jobs map for tracking
	r.activeJobs.Store(response.Job, rule)

	return nil
}

func (r *Runner) checkAWXJobStatus(jobID int) (finished bool, err error) {
	// Get the AWX connection details from the configuration:
	awxAddress := r.config.Address()
	awxProxy := r.config.Proxy()
	awxUser := r.config.User()
	awxPassword := r.config.Password()
	awxCA := r.config.CA()
	awxInsecure := r.config.Insecure()

	// Create the connection to the AWX server:
	connection, err := awx.NewConnectionBuilder().
		URL(awxAddress).
		Proxy(awxProxy).
		Username(awxUser).
		Password(awxPassword).
		CACertificates(awxCA).
		Insecure(awxInsecure).
		Build()
	if err != nil {
		return
	}
	defer connection.Close()

	jobsResource := connection.Jobs()

	jobsResponse, err := jobsResource.Id(jobID).Get().Send()
	if err != nil {
		return
	}

	job := jobsResponse.Job()

	glog.Infof(
		"Job %d status: %s",
		job.Id(),
		job.Status(),
	)

	finished = job.IsFinished()

	return
}
