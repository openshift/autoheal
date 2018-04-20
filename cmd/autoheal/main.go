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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/autoheal/pkg/receiver"
)

var rootCmd = &cobra.Command{
	Use: "autoheal",
	Long: "The auto-heal service receives alert notifications from the Prometheus alert manager, " +
		"decides which healing action to execute based on a set of rules, and executes it. Healing " +
		"actions can be AWX jobs or Kubernetes batch jobs.",
}

func init() {
	rootCmd.AddCommand(receiver.Cmd)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	flag.CommandLine.Parse([]string{})

	// Execute the root command:
	rootCmd.SetArgs(os.Args[1:])
	rootCmd.Execute()
}
