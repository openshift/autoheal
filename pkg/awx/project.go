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

// This file contains the implementation of the project type.

package awx

// Project represents an AWX project.
//
type Project struct {
	id   int
	name string
	scmType string
	scmURL string
	scmBranch string
}

// Id returns the unique identifier of the project.
//
func (p *Project) Id() int {
	return p.id
}

// Name returns the name of the project.
//
func (p *Project) Name() string {
	return p.name
}

// SCMType returns the source code management system type of the project.
//
func (p *Project) SCMType() string {
	return p.scmType
}

// SCMType returns the source code management system URL of the project.
//
func (p *Project) SCMURL() string {
	return p.scmURL
}

// SCMBranch returns the source code management system branch of the project.
//
func (p *Project) SCMBranch() string {
	return p.scmBranch
}
