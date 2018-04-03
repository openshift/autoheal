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
	"log"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/openshift/autoheal/pkg/signals"
	"github.com/spf13/viper"
)

// Values of the command line options:
var (
	serverKubeAddress string
	serverKubeConfig  string
	serverConfigFile  string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the auto-heal server",
	Long:  "Starts the auto-heal server.",
	Run:   serverRun,
}

func init() {
	cobra.OnInitialize(initConfig)
	serverFlags := serverCmd.Flags()
	serverFlags.StringVar(
		&serverKubeConfig,
		"kubeconfig",
		filepath.Join(homedir.HomeDir(), ".kube", "config"),
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
	serverFlags.StringVar(
		&serverConfigFile,
		"config-file",
		"autoheal.yml",
		"The location of the configuration file.",
	)
}

func serverRun(cmd *cobra.Command, args []string) {
	var err error

	// Set up signals so we handle the first shutdown signal gracefully:
	stopCh := signals.SetupSignalHandler()

	// Load the Kubernetes configuration:
	var config *rest.Config
	_, err = os.Stat(serverKubeConfig)
	if os.IsNotExist(err) {
		glog.Infof(
			"The Kubernetes configuration file '%s' doesn't exist, will try to use the "+
				"in-cluster configuration",
			serverKubeConfig,
		)
		config, err = rest.InClusterConfig()
		if err != nil {
			glog.Fatalf(
				"Error loading in-cluster REST client configuration: %s",
				err.Error(),
			)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags(serverKubeAddress, serverKubeConfig)
		if err != nil {
			glog.Fatalf(
				"Error loading REST client configuration from file '%s': %s",
				serverKubeConfig,
				err.Error(),
			)
		}
	}

	// Create the Kuberntes API client:
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Error building Kubernets API client: %s", err.Error())
	}

	// Build the healer:
	healer, err := NewHealerBuilder().
		ConfigFile(serverConfigFile).
		KubernetesClient(k8sClient).
		Build()
	if err != nil {
		glog.Fatalf("Error building healer: %s", err.Error())
	}

	// Run the healer:
	if err = healer.Run(stopCh); err != nil {
		glog.Fatalf("Error running healer: %s", err.Error())
	}
}

func initConfig() {
	viper.SetConfigFile(serverConfigFile)
	if err := viper.ReadInConfig(); err != nil {
		log.Println("Error reading config file:", err)
	}
}
