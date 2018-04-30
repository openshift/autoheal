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

package alertmanager

import (
	"testing"
)

func TestName(t *testing.T) {
	a := Alert{
		Labels: map[string]string{
			"alertname": "foo",
		},
	}
	if a.Name() != "foo" {
		t.Errorf("Expected foo but got %+v", a.Name())
	}
}

func TestNamespace(t *testing.T) {
	a := Alert{}
	if a.Namespace() != "default" {
		t.Errorf("Expected default but got %+v", a.Namespace())
	}
	a.Annotations = map[string]string{"namespace": "foo"}
	if a.Namespace() != "foo" {
		t.Errorf("Expected foo but got %+v", a.Namespace())
	}
	a.Labels = map[string]string{"namespace": "bar"}
	if a.Namespace() != "bar" {
		t.Errorf("Expected bar but got %+v", a.Namespace())
	}
}

func TestHash(t *testing.T) {
	a := Alert{
		Labels: map[string]string{
			"alertname": "foo",
			"test":      "test",
		},
		Annotations: map[string]string{
			"foo":   "bar",
			"test1": "test",
		},
	}
	b := Alert{
		Labels: map[string]string{
			"test":      "test",
			"alertname": "foo",
		},
		Annotations: map[string]string{
			"test1": "test",
			"foo":   "bar",
		},
	}

	aHash := a.Hash()
	bHash := b.Hash()
	if aHash != bHash {
		t.Errorf("Expected same hash, got %+v != %+v", aHash, bHash)
	}
}
