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

// This file contains the implementation of the object template, which is capable of recursively
// iterating an object and replacing all the strings that it contains with the result of evaluating
// them as templates.

package main

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"

	"github.com/golang/glog"
)

// ObjecTemplateBuilder is used to build object template processors. Don't instantiate it directly,
// use the NewObjectTemplateBuilder method instead.
//
type ObjectTemplateBuilder struct {
	// Delimiters:
	right string
	left  string

	// Variables:
	variables map[string]string
}

// NewObjectTemplateBuilder creates a new buildr for object template processors.
//
func NewObjectTemplateBuilder() *ObjectTemplateBuilder {
	b := new(ObjectTemplateBuilder)
	b.right = "}}"
	b.left = "{{"
	return b
}

// Delimiters sets the delimiters that will be used in the templates. By default the delimiters used
// are the ones used in Go templates, {{ and }}. It is convenient to change them when processing
// templates that contain that text, for example Ansible Playbooks.
//
func (b *ObjectTemplateBuilder) Delimiters(left, right string) *ObjectTemplateBuilder {
	b.right = right
	b.left = left
	return b
}

// Variable sets a variable that will be added to all the templates. For example, if the name is
// `labels` and the value is `.Labels` the processor will automatically add this to the beginning of
// all the generated templates:
//
//	{{ $labels := .Labels }}
//
// The syntax of the value is the same syntax used in Go templates for this kind of variables.
//
func (b *ObjectTemplateBuilder) Variable(name, value string) *ObjectTemplateBuilder {
	if b.variables == nil {
		b.variables = make(map[string]string)
	}
	b.variables[name] = value
	return b
}

// Build creates a new template processor with the configuration stored in the builder.
//
func (b *ObjectTemplateBuilder) Build() (t *ObjectTemplate, err error) {
	// Alocate the object:
	t = new(ObjectTemplate)

	// Save the delimiters:
	t.right = b.right
	t.left = b.left

	// Copy the variables:
	t.variables = make(map[string]string)
	for name, value := range b.variables {
		t.variables[name] = value
	}

	return
}

// ObjectTemplate contains the data needed to process the templats inside objects. Don't instantiate
// it directly, use the builder instead. For example:
//
//	template, err := NewObjectTemplateBuilder().
//		Delimiters("[[", "]]").
//		Variable("labels", ".Labels").
//		Variable("annotations", ".Annotations").
//		Build()
//
type ObjectTemplate struct {
	// Delimiters:
	right string
	left  string

	// Variables:
	variables map[string]string
}

// Process iterates the object recursively, and replaces all the fields or items that are strings
// with the result of processing them as templates. The data for the template is taken from the data
// parameter.
//
func (t *ObjectTemplate) Process(object interface{}, data interface{}) error {
	kind := reflect.ValueOf(object).Kind()
	if kind != reflect.Ptr {
		return fmt.Errorf("Bad argument to function. object must be of pointer type, but type is %v", kind)
	}

	if glog.V(2) {
		glog.Infof("Data: %v", data)
	}
	_, err := t.processValue(reflect.ValueOf(object), data)

	return err
}

func (t *ObjectTemplate) processValue(input reflect.Value, data interface{}) (output reflect.Value, err error) {
	output = input
	if output.IsValid() {
		switch output.Kind() {
		case reflect.String:
			text, err := t.processString(output, data)
			if err == nil {
				output = reflect.ValueOf(text)
			}
		case reflect.Array:
			// Not implemented yet.
		case reflect.Slice:
			// Not implemented yet.
		case reflect.Map:
			for _, k := range output.MapKeys() {
				var v reflect.Value
				v, err = t.processValue(output.MapIndex(k), data)
				if err != nil {
					return
				}

				// update the processed vaule in the map
				output.SetMapIndex(k, v)
			}
		case reflect.Struct:
			for i, n := 0, output.NumField(); i < n && err == nil; i++ {
				_, err = t.processValue(output.Field(i), data)
				if err != nil {
					return
				}
			}
		case reflect.Ptr:
			output, err = t.processValue(output.Elem(), data)
		case reflect.Interface:
			output, err = t.processValue(reflect.ValueOf(output.Interface()), data)
		default:
			if glog.V(3) {
				glog.Infof("Unsupported value kind: %v, skipping templating", output.Kind())
			}
		}
	}
	return
}

func (t *ObjectTemplate) processString(value reflect.Value, data interface{}) (text string, err error) {
	// Get the original text:
	text = value.String()
	if glog.V(3) {
		glog.Infof("Original text:\n%s", text)
	}

	// Generate the template text:
	buffer := new(bytes.Buffer)
	for name, value := range t.variables {
		fmt.Fprintf(buffer, "%s $%s := %s %s", t.left, name, value, t.right)
	}
	buffer.WriteString(text)
	text = buffer.String()
	if glog.V(3) {
		glog.Infof("Generated template:\n%s", text)
	}

	// Parse and run the template:
	tmpl, err := template.New("").Delims(t.left, t.right).Parse(text)
	if err != nil {
		return
	}
	buffer.Reset()
	err = tmpl.Execute(buffer, data)
	if err != nil {
		return
	}
	text = buffer.String()
	if glog.V(3) {
		glog.Infof("Generated text:\n%s", text)
	}

	return
}
