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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	monitoring "github.com/jhernand/openshift-monitoring/pkg/apis/monitoring/v1alpha1"
	"github.com/jhernand/openshift-monitoring/pkg/client/informers"
	listers "github.com/jhernand/openshift-monitoring/pkg/client/listers/monitoring/v1alpha1"
	"github.com/jhernand/openshift-monitoring/pkg/client/openshift"
)

// LauncherBuilder is used to create new launchers.
//
type LauncherBuilder struct {
	// Client.
	client openshift.Interface

	// Informer factory.
	informerFactory informers.SharedInformerFactory

	// The locations of the Prometheus binary and configuration files.
	childBinary string
	childConfig string

	// The arguments to pass to the child process.
	childArgs []string
}

// Launcher contains the information needed to receive notifications about changes in the
// Prometheus configuration and to start or reload it when there are changes.
//
type Launcher struct {
	// Clients.
	client openshift.Interface

	// Informer factory.
	informerFactory informers.SharedInformerFactory

	// Listers.
	alertingRuleLister listers.AlertingRuleLister

	// Sync checkers.
	alertingRulesSynced cache.InformerSynced

	// This map stores for each alerting rule name the latest resource version that has been seen by
	// the launcher. This is intended to avoid reloading the Prometheous configuration when repeated
	// update notnifications ara received.
	alertingRuleVersions map[string]string

	// The locations of the Prometheus binary and configuration files.
	childBinary string
	childConfig string

	// The arguments to pass to the child process.
	childArgs []string

	// The child process.
	child *exec.Cmd
}

// NewLauncherBuilder creates a new builder for launchers.
//
func NewLauncherBuilder() *LauncherBuilder {
	b := new(LauncherBuilder)
	b.childBinary = "prometheus"
	b.childConfig = "prometheus.yaml"
	return b
}

// Client sets the OpenShift client that will be used by the launcher.
//
func (b *LauncherBuilder) Client(client openshift.Interface) *LauncherBuilder {
	b.client = client
	return b
}

// InformerFactory sets the OpenShift informer factory that will be used by the launcher.
//
func (b *LauncherBuilder) InformerFactory(factory informers.SharedInformerFactory) *LauncherBuilder {
	b.informerFactory = factory
	return b
}

// Binary sets the location of the child binary.
//
func (b *LauncherBuilder) Binary(location string) *LauncherBuilder {
	b.childBinary = location
	return b
}

// Config sets the location of the child configuration file.
//
func (b *LauncherBuilder) Config(location string) *LauncherBuilder {
	b.childConfig = location
	return b
}

// Args sets the argumets that will be passed to the child.
//
func (b *LauncherBuilder) Args(args []string) *LauncherBuilder {
	b.childArgs = args
	return b
}

// Build creates the launcher using the configuration stored in the builder.
//
func (b *LauncherBuilder) Build() (l *Launcher, err error) {
	// Allocate the launcher:
	l = new(Launcher)

	// Save the references to the Kubernetes and OpenShift clients:
	l.client = b.client

	// Save the details of the child:
	l.childBinary = b.childBinary
	l.childConfig = b.childConfig
	l.childArgs = b.childArgs

	// Initialize the map where we store the latest resource version seen for each alerting rule:
	l.alertingRuleVersions = make(map[string]string)

	// Get references to the shared informers for the kinds of objects that affect the
	// Prometheus configuration:
	alertingRuleInformer := b.informerFactory.Monitoring().V1alpha1().AlertingRules()
	l.alertingRuleLister = alertingRuleInformer.Lister()
	l.alertingRulesSynced = alertingRuleInformer.Informer().HasSynced

	// Set up an event handler for when alerting rules are created, modified or deleted:
	alertingRuleInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				switch change := obj.(type) {
				case *monitoring.AlertingRule:
					glog.Infof(
						"Alerting rule '%s' has been added",
						change.ObjectMeta.Name,
					)
					l.handleAlertingRuleChange(change)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				switch change := newObj.(type) {
				case *monitoring.AlertingRule:
					glog.Infof(
						"Alerting rule '%s' has been updated",
						change.ObjectMeta.Name,
					)
					l.handleAlertingRuleChange(change)
				}
			},
			DeleteFunc: func(obj interface{}) {
				switch change := obj.(type) {
				case *monitoring.AlertingRule:
					glog.Infof(
						"Alerting rule '%s' has been deleted",
						change.ObjectMeta.Name,
					)
					l.handleAlertingRuleChange(change)
				}
			},
		},
	)

	return
}

// Run waits for the informers caches to sync, and then starts the child process. It then waits till
// the stopCh is closed, at which point it asks the child process to finish.
//
func (l *Launcher) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// Wait for the caches to be synced before starting the worker:
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, l.alertingRulesSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	// Build the command line for the child process:
	childArgs := l.childArgs
	if l.childConfig != "" {
		childArgs = append(
			childArgs,
			fmt.Sprintf("--config.file=%s", l.childConfig),
		)
	}

	// Start the child process:
	glog.Infof(
		"Starting child with binary '%s' and arguments '%s'",
		l.childBinary,
		strings.Join(childArgs, " "),
	)
	l.child = exec.Command(l.childBinary, childArgs...)
	l.child.Stdin = os.Stdin
	l.child.Stdout = os.Stdout
	l.child.Stderr = os.Stderr
	err := l.child.Start()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error starting child: %s", err.Error()))
	}
	glog.Infof("Child PID is '%d'", l.child.Process.Pid)

	// Wait till we are requested to stop, and then ask the child process to finish, and wait
	// for it:
	<-stopCh
	glog.Infof("Sending TERM signal to child with PID '%d'", l.child.Process.Pid)
	err = l.child.Process.Signal(syscall.SIGTERM)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error sending TERM signal to child: %s", err.Error()))
	}
	glog.Infof("Waiting for child with PID '%d'", l.child.Process.Pid)
	err = l.child.Wait()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error waiting for child: %s", err.Error()))
	}

	return nil
}

// handleAlertingRuleChange checks if the given alerting rule change has been seen before. If it
// hasn't, then it may be necessary to reload the configuration.
//
func (l *Launcher) handleAlertingRuleChange(change *monitoring.AlertingRule) {
	name := change.ObjectMeta.Name
	current := change.ObjectMeta.ResourceVersion
	previous := l.alertingRuleVersions[name]
	if current != previous {
		l.maybeReload()
		l.alertingRuleVersions[name] = current
	} else {
		glog.Infof(
			"Version '%s' of alerting rule '%s' has been processed before, will ignore it",
			change.ObjectMeta.ResourceVersion,
			change.ObjectMeta.Name,
		)
	}
}

// maybeReload checks if it is necessary to reload the configuration, and if so reloads it.
//
func (l *Launcher) maybeReload() {
	// Retrieve the complete list of alerting rules:
	rules, err := l.alertingRuleLister.List(labels.Everything())
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error listing alerting rules: %s", err.Error()))
	}

	// Generate the YAML text that corresponds to the retrieved list of alerting rules:
	newYaml, err := makeYaml(rules)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error generating alerting rules: %s", err.Error()))
	}
	glog.Infof("Generated alerting rules\n%s", newYaml)

	// Read the current YAML text from the flie, and check if there is any difference with the
	// new one:
	configDir := filepath.Dir(l.childConfig)
	rulesFile := filepath.Join(configDir, "alerting.rules")
	replaceFile := false
	if _, err := os.Stat(rulesFile); err == nil {
		oldYaml, err := ioutil.ReadFile(rulesFile)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Error reading alerting rules file '%s': %s", rulesFile, err.Error()))
		}
		if bytes.Equal(oldYaml, newYaml) {
			glog.Infof(
				"The alerting rules file '%s' doesn't need to be replaced",
				rulesFile,
			)
			replaceFile = false
		} else {
			glog.Infof(
				"The alerting rules file '%s' needs to be replaced",
				rulesFile,
			)
			replaceFile = true
		}
	} else {
		glog.Infof(
			"The alerting rules file '%s' doesn't exist",
			rulesFile,
		)
		replaceFile = true
	}

	// If there are differences, then replace the file and make sure tha the child process is
	// running with the new configuration:
	if replaceFile {
		err = ioutil.WriteFile(rulesFile, newYaml, 0666)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Error writing alerting rules file '%s': %s", rulesFile, err.Error()))
		}
		if l.child != nil {
			glog.Infof("Sending HUP signal to PID '%d'", l.child.Process.Pid)
			err := l.child.Process.Signal(syscall.SIGHUP)
			if err != nil {
				runtime.HandleError(fmt.Errorf("Error sending signal to child: %s", err.Error()))
			}
			glog.Infof("Sent HUP signal to PID '%d'", l.child.Process.Pid)
		}
	}
}
