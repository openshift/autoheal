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

// This example shows how to retrieve a specific project.
//
// Use the following command to build and run it with all the debug output sent to the standard
// error output:
//
//	go run get_project_by_name.go \
//		-url "https://awx.example.com/api" \
//		-username "admin" \
//		-password "..." \
//		-ca-file "ca.pem" \
//		-name "My project" \
//		-logtostderr \
//		-v=2

package main

import (
	"flag"
	"fmt"

	"github.com/openshift/autoheal/pkg/awx"
)

var (
	url      string
	username string
	password string
	proxy    string
	insecure bool
	caFile   string
	name     string
)

func init() {
	flag.StringVar(&url, "url", "https://awx.example.com/api", "API URL.")
	flag.StringVar(&username, "username", "admin", "API user name.")
	flag.StringVar(&password, "password", "", "API user password.")
	flag.StringVar(&proxy, "proxy", "", "API proxy URL.")
	flag.BoolVar(&insecure, "insecure", false, "Don't verify server certificate.")
	flag.StringVar(&caFile, "ca-file", "", "Trusted CA certificates.")
	flag.StringVar(&name, "name", "", "The name of the project.")
}

func main() {
	// Parse the command line:
	flag.Parse()

	// Connect to the server, and remember to close the connection:
	connection, err := awx.NewConnectionBuilder().
		Url(url).
		Username(username).
		Password(password).
		Proxy(proxy).
		CAFile(caFile).
		Insecure(insecure).
		Build()
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	// Find the resource that manages the collection of projects:
	projectsResource := connection.Projects()

	// Send the request to get the list of projects that have a given name:
	getProjectsResponse, err := projectsResource.Get().
		Filter("name", name).
		Send()
	if err != nil {
		panic(err)
	}

	// Print the results:
	projects := getProjectsResponse.Results()
	if len(projects) == 0 {
		fmt.Printf("There is no project named '%s'.\n", name)
	} else {
		for _, project := range projects {
			fmt.Printf("Id is '%d'.\n", project.Id())
			fmt.Printf("Name is '%s'.\n", project.Name())
			fmt.Printf("SCM type is '%s'.\n", project.SCMType())
			fmt.Printf("SCM URL is '%s'.\n", project.SCMURL())
			fmt.Printf("SCM branch is '%s'.\n", project.SCMBranch())
		}
	}
}
