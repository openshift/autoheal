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
	"testing"
)

type TemplateTestDataNested struct {
	StructKey string
}

type TemplateTestData struct {
	Map    map[string]string
	Str    string
	Struct TemplateTestDataNested
}

func TestProcess(t *testing.T) {
	template, err := NewObjectTemplateBuilder().
		Delimiters("[", "]").
		Variable("map", ".Map").
		Variable("str", ".Str").
		Variable("struct", ".Struct").
		Build()

	if err != nil {
		t.Errorf("Error building ObjectTemplate: %v", err)
	}

	params := TemplateTestData{
		Map:    map[string]string{"mapkey": "mapvalue"},
		Str:    "This is a string",
		Struct: TemplateTestDataNested{StructKey: "foo"},
	}

	testProcessStringInput(t, template, params)
	testProcessStructInput(t, template, params)
	testProcessMapInput(t, template, params)

}

// check basic string templating:
func testProcessStringInput(t *testing.T, template *ObjectTemplate, params TemplateTestData) {
	input := "Test [ $foo ] test [ $bar ]"
	err := template.Process(&input, params)
	if err != nil {
		t.Errorf("Error processing template: %v", err)
	}

	if input != "Test [ $foo ] test [ $bar ]" {
		t.Errorf("Input changed even though it didn't match template! %v", input)
	}

	input = "str=[ $str ]"
	err = template.Process(&input, params)
	if err != nil {
		t.Errorf("Error processing template: %v", err)
	}

	expected := "str=This is a string"

	if input != expected {
		t.Errorf("Unexpected template result - expected '%v', got '%v'", expected, input)
	}
}

// check struct templating:
func testProcessStructInput(t *testing.T, template *ObjectTemplate, params TemplateTestData) {
	type TestStruct struct {
		Templated  string
		Unchanged  string
		unsettable string //Unexported fields are unsettable via reflection
	}
	input := TestStruct{
		Templated:  "str=[ $str ]",
		Unchanged:  "Test [ $foo ] test [ $bar ]",
		unsettable: "str=[ $str ]",
	}

	err := template.Process(&input, params)
	if err != nil {
		t.Errorf("Error processing template: %v", err)
	}

	expected := TestStruct{
		Templated:  "str=This is a string",
		Unchanged:  "Test [ $foo ] test [ $bar ]",
		unsettable: "str=[ $str ]",
	}

	if input != expected {
		t.Errorf("Unexpected template result - expected '%v', got '%v'", expected, input)
	}

}

// check map templating:
func testProcessMapInput(t *testing.T, template *ObjectTemplate, params TemplateTestData) {
	input := map[string]string{
		"a": "Test [ $foo ] test [ $bar ]",
		"b": "Map=[ $map ], str=[ $str ], struct=[ $struct ]",
	}
	err := template.Process(&input, params)
	if err != nil {
		t.Errorf("Error processing template: %v", err)
	}

	if input["a"] != "Test [ $foo ] test [ $bar ]" {
		t.Errorf("Input changed even though it didn't match template! %v", input["a"])
	}

	expected := "Map=map[mapkey:mapvalue], str=This is a string, struct={foo}"
	if input["b"] != expected {
		t.Errorf("Unexpected template result - expected '%v', got '%v'", expected, input["b"])
	}
}
