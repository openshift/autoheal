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
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/jhernand/openshift-monitoring/pkg/client/informers"
	"github.com/jhernand/openshift-monitoring/pkg/client/openshift"
	"github.com/jhernand/openshift-monitoring/pkg/signals"
)

// Values of the command line options:
var (
	kubeAddress string
	kubeConfig  string
)

func main() {
	var err error

	// Define the command line options:
	flag.StringVar(
		&kubeConfig,
		"kubeconfig",
		filepath.Join(homedir.HomeDir(), ".kube", "config"),
		"Path to a Kubernetes client configuration file. Only required when running "+
			"outside of a cluster.",
	)
	flag.StringVar(
		&kubeAddress,
		"master",
		"",
		"The address of the Kubernetes API server. Overrides any value in the Kubernetes "+
			"configuration file. Only required when running outside of a cluster.",
	)

	// Parse the command line:
	flag.Parse()

	// Set up signals so we handle the first shutdown signal gracefully:
	stopCh := signals.SetupSignalHandler()

	// Load the Kubernetes configuration:
	var config *rest.Config
	_, err = os.Stat(kubeConfig)
	if os.IsNotExist(err) {
		glog.Infof(
			"The Kubernetes configuration file '%s' doesn't exist, will try to use the "+
				"in-cluster configuration",
			kubeConfig,
		)
		config, err = rest.InClusterConfig()
		if err != nil {
			glog.Fatalf(
				"Error loading in-cluster REST client configuration: %s",
				err.Error(),
			)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags(kubeAddress, kubeConfig)
		if err != nil {
			glog.Fatalf(
				"Error loading REST client configuration from file '%s': %s",
				kubeConfig,
				err.Error(),
			)
		}
	}

	// Create the Kuberntes API client:
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Error building Kubernets API client: %s", err.Error())
	}

	// Create the OpenShift API client:
	osClient, err := openshift.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Error building OpenShift API client: %s", err.Error())
	}

	// Create an informer factory that will create informers that sync every 5 minutes:
	informerFactory := informers.NewSharedInformerFactory(osClient, 5*time.Minute)

	// Build the healer:
	healer, err := NewHealerBuilder().
		KubernetesClient(k8sClient).
		OpenShiftClient(osClient).
		InformerFactory(informerFactory).
		Build()
	if err != nil {
		glog.Fatalf("Error building healer: %s", err.Error())
	}

	// Start the informer factory:
	go informerFactory.Start(stopCh)

	// Run the healer:
	if err = healer.Run(stopCh); err != nil {
		glog.Fatalf("Error running healer: %s", err.Error())
	}
}
