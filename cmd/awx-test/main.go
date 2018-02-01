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
	"fmt"

	"github.com/jhernand/openshift-monitoring/pkg/awx"
)

func main() {
	var err error

	// Parse the command line:
	flag.Parse()

	// Create the connection to the AWX server:
	connection, err := awx.NewConnectionBuilder().
		Url("https://tower.yellow/api/").
		Proxy("http://server0.mad.redhat.com:3128").
		Username("admin").
		Password("redhat123").
		Insecure(true).
		Build()
	if err != nil {
		panic(err)
		return
	}
	defer connection.Close()

	{
		// List the job templates:
		response, err := connection.JobTemplates().Get().
			Filter("name", "Demo Job Template").
			Send()
		if err != nil {
			panic(err)
		}
		fmt.Printf("count: %d\n", response.Count())

		// Print the names of the job templates:
		for _, result := range response.Results() {
			fmt.Printf("job id: %s\n", result.Id())
			fmt.Printf("job name: %s\n", result.Name())
		}
	}

	{
		// Send a request to retrieve an specific job template:
		response, err := connection.JobTemplates().Id("5").Get().Send()
		if err != nil {
			panic(err)
		}
		fmt.Printf("jobTemplate: %s\n", response.Result().Name())
	}

	{
		// Send a request to get the launch data for a job template:
		obj, err := connection.JobTemplates().Id("5").Launch().Get().Send()
		if err != nil {
			panic(err)
		}
		fmt.Printf("job name: %s\n", obj.JobTemplateData().Name())
		fmt.Printf("job id: %d\n", obj.JobTemplateData().Id())

		response, err := connection.JobTemplates().Id("5").Launch().Post().
			ExtraVars("myvar: myvalue").
			Send()
		if err != nil {
			panic(err)
		}
		fmt.Printf("launch response: %s\n", response)
	}

	fmt.Printf("Bye!\n")
}
