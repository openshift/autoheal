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
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/golang/glog"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	awx "github.com/jhernand/openshift-monitoring/pkg/awx"
	genericinf "github.com/jhernand/openshift-monitoring/pkg/client/informers"
	typedinf "github.com/jhernand/openshift-monitoring/pkg/client/informers/monitoring/v1alpha1"
	openshift "github.com/jhernand/openshift-monitoring/pkg/client/openshift"
)

// HealerBuilder is used to create new healers.
//
type HealerBuilder struct {
	// Clients.
	k8sClient kubernetes.Interface
	osClient  openshift.Interface

	// Informer factory.
	informerFactory genericinf.SharedInformerFactory
}

// Healer contains the information needed to receive notifications about changes in the
// Prometheus configuration and to start or reload it when there are changes.
//
type Healer struct {
	// Client.
	k8sClient kubernetes.Interface
	osClient  openshift.Interface

	// Informer factory.
	informerFactory genericinf.SharedInformerFactory

	// Informers.
	alertInformer       typedinf.AlertInformer
	healingRuleInformer typedinf.HealingRuleInformer
}

// NewHealerBuilder creates a new builder for healers.
//
func NewHealerBuilder() *HealerBuilder {
	b := new(HealerBuilder)
	return b
}

// KubernetesClient sets the Kubernetes client that will be used by the healer.
//
func (b *HealerBuilder) KubernetesClient(client kubernetes.Interface) *HealerBuilder {
	b.k8sClient = client
	return b
}

// OpenShiftClient sets the OpenShift client that will be used by the healer.
//
func (b *HealerBuilder) OpenShiftClient(client openshift.Interface) *HealerBuilder {
	b.osClient = client
	return b
}

// InformerFactory sets the OpenShift informer factory that will be used by the healer.
//
func (b *HealerBuilder) InformerFactory(factory genericinf.SharedInformerFactory) *HealerBuilder {
	b.informerFactory = factory
	return b
}

// Build creates the healer using the configuration stored in the builder.
//
func (b *HealerBuilder) Build() (h *Healer, err error) {
	// Allocate the healer:
	h = new(Healer)

	// Save the references to the clients:
	h.k8sClient = b.k8sClient
	h.osClient = b.osClient

	// Save the reference to the informer factory:
	h.informerFactory = b.informerFactory

	return
}

// Run waits for the informers caches to sync, and then starts the healer.
//
func (h *Healer) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// Create the informers:
	h.alertInformer = h.informerFactory.Monitoring().V1alpha1().Alerts()
	h.healingRuleInformer = h.informerFactory.Monitoring().V1alpha1().HealingRules()

	// Wait for the caches to be synced before starting the worker:
	glog.Info("Waiting for informer caches to sync")
	alertsSynced := h.alertInformer.Informer().HasSynced
	healingRulesSynced := h.healingRuleInformer.Informer().HasSynced
	if ok := cache.WaitForCacheSync(stopCh, alertsSynced, healingRulesSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	// Set up an event handler for when alerts are created, modified or deleted:
	h.alertInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				switch change := obj.(type) {
				case *monitoring.Alert:
					glog.Infof(
						"Alert '%s' has been added",
						change.ObjectMeta.Name,
					)
					h.handleAlertChange(change)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				switch change := newObj.(type) {
				case *monitoring.Alert:
					glog.Infof(
						"Alert '%s' has been updated",
						change.ObjectMeta.Name,
					)
					h.handleAlertChange(change)
				}
			},
			DeleteFunc: func(obj interface{}) {
				switch change := obj.(type) {
				case *monitoring.Alert:
					glog.Infof(
						"Alert '%s' has been deleted",
						change.ObjectMeta.Name,
					)
					h.handleAlertChange(change)
				}
			},
		},
	)

	// Wait till we are requested to stop:
	<-stopCh

	return nil
}

// handleAlertChange checks if the given alert change requires starting a healing process.
//
func (h *Healer) handleAlertChange(alert *monitoring.Alert) {
	// Load the healing rules:
	rules, err := h.healingRuleInformer.Lister().List(labels.Everything())
	if err != nil {
		glog.Info("Can't load healing rules: %s", err.Error())
	}

	// Find the rules that are activated for the alert:
	activated := make([]*monitoring.HealingRule, 0)
	for _, rule := range rules {
		if h.checkConditions(rule, alert) {
			glog.Infof(
				"Healing rule '%s' matches alert '%s'",
				rule.ObjectMeta.Name,
				alert.ObjectMeta.Name,
			)
			activated = append(activated, rule)
		}
	}
	if len(activated) == 0 {
		glog.Infof("No healing rule matches alert '%s'", alert.ObjectMeta.Name)
		return
	}

	// Execute the actions of the activated rules:
	for _, rule := range activated {
		h.runActions(rule, alert)
	}
}

func (h *Healer) checkConditions(rule *monitoring.HealingRule, alert *monitoring.Alert) bool {
	glog.Infof(
		"Checking conditions of rule '%s' for alert '%s'",
		rule.ObjectMeta.Name,
		alert.ObjectMeta.Name,
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

func (h *Healer) checkCondition(condition *monitoring.HealingCondition, alert *monitoring.Alert) bool {
	matched, err := regexp.MatchString(condition.Alert, alert.ObjectMeta.Name)
	if err != nil {
		glog.Errorf(
			"Error while checking if alert name '%s' matches pattern '%s': %s",
			alert.ObjectMeta.Name,
			condition.Alert,
			err.Error(),
		)
		matched = false
	}
	return matched
}

func (h *Healer) runActions(rule *monitoring.HealingRule, alert *monitoring.Alert) {
	if rule.Spec.Actions != nil && len(rule.Spec.Actions) > 0 {
		glog.Infof(
			"Running actions of healing rule '%s' for alert '%s'",
			rule.ObjectMeta.Name,
			alert.ObjectMeta.Name,
		)
		for i := 0; i < len(rule.Spec.Actions); i++ {
			h.runAction(rule, &rule.Spec.Actions[i], alert)
		}
	} else {
		glog.Warningf(
			"Healing rule '%s' has no actions, will have no effect on alert '%s'",
			rule.ObjectMeta.Name,
			alert.ObjectMeta.Name,
		)
	}
}

func (h *Healer) runAction(rule *monitoring.HealingRule, action *monitoring.HealingAction, alert *monitoring.Alert) {
	if action.AWXJob != nil {
		h.runAWXJob(rule, action.AWXJob, alert)
	} else if action.BatchJob != nil {
		h.runBatchJob(rule, action.BatchJob, alert)
	} else {
		glog.Warningf(
			"There are no action details, rule '%s' will have no effect on alert '%s'",
			rule.ObjectMeta.Name,
			alert.ObjectMeta.Name,
		)
	}
}

func (h *Healer) runAWXJob(rule *monitoring.HealingRule, action *monitoring.AWXJobAction, alert *monitoring.Alert) {
	glog.Infof(
		"Running AWX job from project '%s' and template '%s' to heal alert '%s'",
		action.Project,
		action.Template,
		alert.ObjectMeta.Name,
	)

	// Load the AWX credentials:
	secret := action.SecretRef
	if secret == nil {
		glog.Errorf(
			"The secret containing the AWX credentials hasn't been specified",
		)
		return
	}
	username, password, err := h.loadAWXSecret(rule, secret)
	if err != nil {
		glog.Errorf(
			"Can't load AWX credentials from secret '%s': %s",
			secret.Name,
			err.Error(),
		)
		return
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
		glog.Errorf("Can't create connection to AWX server: %s", err.Error())
		return
	}
	defer connection.Close()

	// Retrieve the job template:
	templatesResource := connection.JobTemplates()
	templatesResponse, err := templatesResource.Get().
		Filter("project__name", action.Project).
		Filter("name", action.Template).
		Send()
	if err != nil {
		glog.Errorf(
			"Can't retrieve AWX job templates named '%s' for project '%s': %s",
			action.Template,
			action.Project,
			err.Error(),
		)
		return
	}
	if templatesResponse.Count() == 0 {
		glog.Errorf(
			"There are no AWX job templates named '%s' for project '%s'",
			action.Template,
			action.Project,
		)
		return
	}

	// Launch the jobs:
	for _, template := range templatesResponse.Results() {
		h.launchAWXJob(connection, template, alert)
	}
}

func (h *Healer) launchAWXJob(connection *awx.Connection, template *awx.JobTemplate, alert *monitoring.Alert) {
	// Convert the alert to a JSON document in order to pass it as the content of the extra
	// variables of the AWX job:
	alertJSON, err := json.Marshal(alert)
	if err != nil {
		glog.Errorf(
			"Can't convert alert '%s' to JSON: %s'",
			alert.ObjectMeta.Name,
			err.Error(),
		)
	}

	// Send the request to launch the job:
	templateId := template.Id()
	templateName := template.Name()
	launchResource := connection.JobTemplates().Id(templateId).Launch()
	_, err = launchResource.Post().
		ExtraVars(string(alertJSON)).
		Send()
	if err != nil {
		glog.Errorf(
			"Can't send request to launch job from template '%s': %s",
			templateName,
			err.Error(),
		)
	}
	glog.Infof(
		"Request to launch AWX job from template '%s' has been sent",
		templateName,
	)
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

func (h *Healer) runBatchJob(rule *monitoring.HealingRule, job *batch.Job, alert *monitoring.Alert) {
	glog.Infof(
		"Running batch job '%s' to heal alert '%s'",
		job.ObjectMeta.Name,
		alert.ObjectMeta.Name,
	)

	// The name of the job is mandatory:
	name := job.ObjectMeta.Name
	if name == "" {
		glog.Errorf(
			"Can't create job for rule '%s', the name hasn't been specified",
			rule.ObjectMeta.Name,
		)
		return
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
			alert.ObjectMeta.Name,
		)
	} else if err != nil {
		glog.Errorf(
			"Can't create batch job '%s' to heal alert '%s'",
			job.ObjectMeta.Name,
			alert.ObjectMeta.Name,
		)
	} else {
		glog.Infof(
			"Batch job '%s' to heal alert '%s' has been created",
			job.ObjectMeta.Name,
			alert.ObjectMeta.Name,
		)
	}
}
