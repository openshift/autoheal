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

package memory

import (
	"testing"
	"time"

	monitoring "github.com/openshift/autoheal/pkg/apis/monitoring/v1alpha1"
)

func TestExisting(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	action := &monitoring.AWXJobAction{
		Template: "My template",
	}
	memory.Add(action)
	if !memory.Has(action) {
		t.Fail()
	}
}

func TestNotExisting(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	action := &monitoring.AWXJobAction{
		Template: "My template",
	}
	if memory.Has(action) {
		t.Fail()
	}
}

func TestSameTemplate(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	first := &monitoring.AWXJobAction{
		Template: "My template",
	}
	second := &monitoring.AWXJobAction{
		Template: "My template",
	}
	memory.Add(first)
	if !memory.Has(first) {
		t.Fail()
	}
	if !memory.Has(second) {
		t.Fail()
	}
}

func TestDifferentTemplate(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	first := &monitoring.AWXJobAction{
		Template: "My template",
	}
	second := &monitoring.AWXJobAction{
		Template: "Your template",
	}
	memory.Add(first)
	if !memory.Has(first) {
		t.Fail()
	}
	if memory.Has(second) {
		t.Fail()
	}
}

func TestSameVars(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	first := &monitoring.AWXJobAction{
		ExtraVars: `{
			"myvar": "myvalue"
		}`,
	}
	second := &monitoring.AWXJobAction{
		ExtraVars: `{
			"myvar": "myvalue"
		}`,
	}
	memory.Add(first)
	if !memory.Has(first) {
		t.Fail()
	}
	if !memory.Has(second) {
		t.Fail()
	}
}

func TestDifferentVars(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	first := &monitoring.AWXJobAction{
		ExtraVars: `{
			"myvar": "myvalue"
		}`,
	}
	second := &monitoring.AWXJobAction{
		ExtraVars: `{
			"yourvar": "yourvalue"
		}`,
	}
	memory.Add(first)
	if !memory.Has(first) {
		t.Fail()
	}
	if memory.Has(second) {
		t.Fail()
	}
}

func TestExpired(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	action := &monitoring.AWXJobAction{
		Template: "My template",
	}
	memory.Add(action)
	if !memory.Has(action) {
		t.Fail()
	}
	time.Sleep(2 * time.Millisecond)
	if memory.Has(action) {
		t.Fail()
	}
}

func TestUpdate(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	action := &monitoring.AWXJobAction{
		Template: "My template",
	}
	memory.Add(action)
	time.Sleep(900 * time.Nanosecond)
	memory.Add(action)
	time.Sleep(200 * time.Nanosecond)
	if !memory.Has(action) {
		t.Fail()
	}
}

func TestLen(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	first := &monitoring.AWXJobAction{
		Template: "My template",
	}
	memory.Add(first)
	if memory.Len() != 1 {
		t.Fail()
	}
	second := &monitoring.AWXJobAction{
		Template: "Your template",
	}
	memory.Add(second)
	if memory.Len() != 2 {
		t.Fail()
	}
}

func TestLenExpired(t *testing.T) {
	memory := makeMemory(t, 1*time.Millisecond)
	action := &monitoring.AWXJobAction{
		Template: "My template",
	}
	memory.Add(action)
	time.Sleep(2 * time.Millisecond)
	if memory.Len() != 0 {
		t.Fail()
	}
}

func makeMemory(t *testing.T, duration time.Duration) *ShortTermMemory {
	memory, err := NewShortTermMemoryBuilder().
		Duration(duration).
		Build()
	if err != nil {
		t.Error(err)
	}
	return memory
}
