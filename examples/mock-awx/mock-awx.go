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

// Package main contains a mock AWX server for autoheal development
//
package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func logRequest(r *http.Request) {
	log.Printf("%s: %s", r.Method, r.URL.Path)

	if r.Method == "POST" {
		// Read body
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err == nil {
			log.Print("Request body:")
			log.Print(string(body))
		}
	}
}

func handlerPostLaunch(w http.ResponseWriter, r *http.Request) {
	POST_JOB := `{
      "job": 4
    }`

	log.Print("Request: launch a job template.")

	logRequest(r)
	w.Write([]byte(POST_JOB))
}

func handlerGetJobList(w http.ResponseWriter, r *http.Request) {
	GET_TEMPLATES := `{
      "count": 1,
      "next": null,
      "previous": null,
      "results": [{"id": 1}]
    }`

	log.Print("Request: list job templates.")

	logRequest(r)
	w.Write([]byte(GET_TEMPLATES))
}

func handler(w http.ResponseWriter, r *http.Request) {
	EMPTY_JSON := `{}`

	log.Print("Request: un handled request.")

	logRequest(r)
	w.Write([]byte(EMPTY_JSON))
}

func main() {
	http.HandleFunc("/api/v2/job_templates/1/launch/", handlerPostLaunch)
	http.HandleFunc("/api/v2/job_templates/", handlerGetJobList)
	http.HandleFunc("/", handler)

	log.Print("Running mock AWX server on port 8080.")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
