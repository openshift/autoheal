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
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/openshift/autoheal/pkg/metrics"
	"github.com/openshift/autoheal/pkg/signals"
)

// Values of the command line options:
var (
	serverKubeAddress string
	serverKubeConfig  string
	serverConfigFiles []string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the auto-heal server",
	Long:  "Starts the auto-heal server.",
	Run:   serverRun,
}

func init() {
	serverFlags := serverCmd.Flags()
	serverFlags.StringVar(
		&serverKubeConfig,
		"kubeconfig",
		"",
		"Path to a Kubernetes client configuration file. Only required when running "+
			"outside of a cluster.",
	)
	serverFlags.StringVar(
		&serverKubeAddress,
		"master",
		"",
		"The address of the Kubernetes API server. Overrides any value in the Kubernetes "+
			"configuration file. Only required when running outside of a cluster.",
	)
	serverFlags.StringSliceVar(
		&serverConfigFiles,
		"config-file",
		[]string{"autoheal.yml"},
		"The location of the configuration file. Can be used multiple times to specify "+
			"multiple configuration files or directories. They will be loaded in the "+
			"same order that they appear in the command line. When the value is a "+
			"directory all the files inside whose names end in .yml or .yaml will be "+
			"loaded, in alphabetical order.",
	)
}

func kubeConfigPath(serverKubeConfig string) (kubeConfig string, err error) {
	// The loading order follows these rules:
	// 1. If the –kubeconfig flag is set,
	// then only that file is loaded. The flag may only be set once.
	// 2. If $KUBECONFIG environment variable is set, use it.
	// 3. Otherwise, ${HOME}/.kube/config is used.
	var ok bool

	// Get the config file path
	if serverKubeConfig != "" {
		kubeConfig = serverKubeConfig
	} else {
		if kubeConfig, ok = os.LookupEnv("KUBECONFIG"); ok != true {
			kubeConfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}
	}

	// Check config file:
	fInfo, err := os.Stat(kubeConfig)
	if os.IsNotExist(err) {
		// NOTE: If config file does not exist, assume using pod configuration.
		err = fmt.Errorf("The Kubernetes configuration file '%s' doesn't exist", kubeConfig)
		kubeConfig = ""
		return
	}

	// Check error codes.
	if fInfo.IsDir() {
		err = fmt.Errorf("The Kubernetes configuration path '%s' is a direcory", kubeConfig)
		return
	}
	if os.IsPermission(err) {
		err = fmt.Errorf("Can't open Kubernetes configuration file '%s'", kubeConfig)
		return
	}

	return
}

func serverRun(cmd *cobra.Command, args []string) {
	// Set up signals so we handle the first shutdown signal gracefully:
	stopCh := signals.SetupSignalHandler()

	// Load the Kubernetes configuration:
	var config *rest.Config

	kubeConfig, err := kubeConfigPath(serverKubeConfig)
	if err == nil {
		// If error is nil, we have a valid kubeConfig file:
		config, err = clientcmd.BuildConfigFromFlags(serverKubeAddress, kubeConfig)
		if err != nil {
			glog.Fatalf(
				"Error loading REST client configuration from file '%s': %s",
				kubeConfig, err,
			)
		}
	} else if kubeConfig == "" {
		glog.Infof("Info: %s", err)

		// If kubeConfig is "", file is missing, in this case we will
		// try to use in-cluster configuration.
		glog.Info("Try to use the in-cluster configuration")
		config, err = rest.InClusterConfig()

		// Catch in-cluster configuration error:
		if err != nil {
			glog.Fatalf("Error loading in-cluster REST client configuration: %s", err)
		}
	} else {
		// Catch all errors:
		glog.Fatalf("Error: %s", err)
	}

	// Create the Kuberntes API client:
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Error building Kubernets API client: %s", err.Error())
	}

	// Build the healer:
	healer, err := NewHealerBuilder().
		ConfigFiles(serverConfigFiles).
		KubernetesClient(k8sClient).
		Build()
	if err != nil {
		glog.Fatalf("Error building healer: %s", err.Error())
	}

	// Register exported metrics:
	metrics.InitExportedMetrics()

	// Run the healer:
	if err = healer.Run(stopCh); err != nil {
		glog.Fatalf("Error running healer: %s", err.Error())
	}
}
