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

package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFiles(t *testing.T) {
	l := NewBuilder()

	data0 := `
      awx:
        address: "http://test_address.com"`

	data1 := `
      awx:
        proxy: "http://test-proxy.com:1234"`

	tempDir, _ := ioutil.TempDir("", "test_dir")

	file0, _ := ioutil.TempFile(tempDir, "test_file0")
	file1, _ := ioutil.TempFile(tempDir, "test_file1")

	file0.WriteString(data0)
	file1.WriteString(data1)

	defer file0.Close()
	defer file1.Close()

	defer os.RemoveAll(tempDir)

	l.Files([]string{file0.Name(), file1.Name()})
	cfg, err := l.Build()
	if err != nil {
		t.Errorf("An error occured! %s", err)
	}
	defer cfg.ShutDown()

	expected := &Config{
		awx: &AWXConfig{
			address: "http://test_address.com",
			proxy:   "http://test-proxy.com:1234",
			jobStatusCheckInterval: 5 * time.Minute,
			ca: new(bytes.Buffer),
		},
		throttling: &ThrottlingConfig{
			interval: 1 * time.Hour,
		},
		rules: &RulesConfig{},
	}

	if err != nil {
		t.Errorf("An error occured! %s", err)
	}

	if !reflect.DeepEqual(cfg.awx, expected.awx) {
		t.Errorf("Expected %+v but got %+v", expected.awx, cfg.awx)
	}

	if !reflect.DeepEqual(cfg.throttling, expected.throttling) {
		t.Errorf("Expected %+v but got %+v", expected.throttling, cfg.throttling)
	}

	if !reflect.DeepEqual(cfg.rules.rules, expected.rules.rules) {
		t.Errorf("Expected %+v but got %+v", expected.rules.rules, cfg.rules.rules)
	}
}

func TestLoadFile(t *testing.T) {
	filename := "test_config"
	file, _ := ioutil.TempFile("", filename)

	defer file.Close()
	defer os.Remove(file.Name())

	configsTest := []struct {
		configString string
		expected     *Config
	}{
		{
			configString: "",
			expected: &Config{
				awx: &AWXConfig{
					jobStatusCheckInterval: time.Duration(5) * time.Minute,
					ca: new(bytes.Buffer),
				},
				throttling: &ThrottlingConfig{
					interval: time.Duration(1) * time.Hour,
				},
				rules: &RulesConfig{},
			},
		},
		{
			configString: `
               awx:
                 address: https://my-awx.example.com/api
                 proxy: http://my-proxy.example.com:3128
               rules:
               - metadata:
                   name: start-node
                 labels:
                   alertname: "NodeDown"
                 awxJob:
                   template: "Start node"`,
			expected: &Config{
				awx: &AWXConfig{
					address: "https://my-awx.example.com/api",
					proxy:   "http://my-proxy.example.com:3128",
					jobStatusCheckInterval: time.Duration(5) * time.Minute,
					ca: new(bytes.Buffer),
				},
				throttling: &ThrottlingConfig{
					interval: time.Duration(1) * time.Hour,
				},
				rules: &RulesConfig{
					rules: []*autoheal.HealingRule{
						{
							ObjectMeta: meta.ObjectMeta{
								Name: "start-node",
							},
							Labels: map[string]string{
								"alertname": "NodeDown",
							},
							AWXJob: &autoheal.AWXJobAction{
								Template: "Start node",
							},
						},
					},
				},
			},
		},
		{
			configString: `
               awx:
                 address: https://my-awx.example.com/api
                 proxy: http://my-proxy.example.com:3128
                 project: "Test Project"
                 jobStatusCheckInterval: 3m
               throttling:
                 interval: 1h
               rules:
               - metadata:
                   name: start-node
                 labels:
                   alertname: "NodeDown"
                 awxJob:
                   template: "Start node"`,
			expected: &Config{
				awx: &AWXConfig{
					address:                "https://my-awx.example.com/api",
					proxy:                  "http://my-proxy.example.com:3128",
					project:                "Test Project",
					jobStatusCheckInterval: time.Duration(3) * time.Minute,
					ca: new(bytes.Buffer),
				},
				throttling: &ThrottlingConfig{
					interval: time.Duration(1) * time.Hour,
				},
				rules: &RulesConfig{
					rules: []*autoheal.HealingRule{
						{
							ObjectMeta: meta.ObjectMeta{
								Name: "start-node",
							},
							Labels: map[string]string{
								"alertname": "NodeDown",
							},
							AWXJob: &autoheal.AWXJobAction{
								Template: "Start node",
							},
						},
					},
				},
			},
		},
		{
			configString: `
             awx:
               address: https://my-awx.example.com/api
               proxy: http://my-proxy.example.com:3128
               project: "Test Project"
               jobStatusCheckInterval: 3m
             throttling:
               interval: 1h
             rules:
             - metadata:
                 name: start-node
               labels:
                 alertname: "NodeDown"
               awxJob:
                 template: "Start node"
             - metadata:
                 name: say-hello
               labels:
                 alertname: "NewFriend"
               batchJob:
                 apiVersion: batch/v1
                 kind: Job
                 metadata:
                   namespace: default
                   name: hello
                 spec:
                   template:
                     spec:
                       containers:
                       - name: python
                         image: python
                         command:
                         - python
                         - -c
                         - print("Hello {{ $labels.name }}!")
                       restartPolicy: Never`,
			expected: &Config{
				awx: &AWXConfig{
					address:                "https://my-awx.example.com/api",
					proxy:                  "http://my-proxy.example.com:3128",
					project:                "Test Project",
					jobStatusCheckInterval: time.Duration(3) * time.Minute,
					ca: new(bytes.Buffer),
				},
				throttling: &ThrottlingConfig{
					interval: time.Duration(1) * time.Hour,
				},
				rules: &RulesConfig{
					rules: []*autoheal.HealingRule{
						{
							ObjectMeta: meta.ObjectMeta{
								Name: "start-node",
							},
							Labels: map[string]string{
								"alertname": "NodeDown",
							},
							AWXJob: &autoheal.AWXJobAction{
								Template: "Start node",
							},
						},
						{
							ObjectMeta: meta.ObjectMeta{
								Name: "say-hello",
							},
							Labels: map[string]string{
								"alertname": "NewFriend",
							},
							BatchJob: &batch.Job{
								TypeMeta: meta.TypeMeta{
									APIVersion: "batch/v1",
									Kind:       "Job",
								},
								ObjectMeta: meta.ObjectMeta{
									Namespace: "default",
									Name:      "hello",
								},
								Spec: batch.JobSpec{
									Template: core.PodTemplateSpec{
										Spec: core.PodSpec{
											Containers: []core.Container{
												{
													Name:    "python",
													Image:   "python",
													Command: []string{"python", "-c", `print("Hello {{ $labels.name }}!")`},
												},
											},
											RestartPolicy: "Never",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	l := NewBuilder()
	l.File(file.Name())

	for _, test := range configsTest {
		func() {
			file.WriteAt([]byte(test.configString), 0)
			cfg, err := l.Build()
			if err != nil {
				t.Errorf("An error occured! %s", err)
			}
			defer cfg.ShutDown()

			if !reflect.DeepEqual(cfg.awx, test.expected.awx) {
				t.Errorf("Expected %+v but got %+v", test.expected.awx, cfg.awx)
			}

			if !reflect.DeepEqual(cfg.throttling, test.expected.throttling) {
				t.Errorf("Expected %+v but got %+v", test.expected.throttling, cfg.throttling)
			}

			if !reflect.DeepEqual(cfg.rules.rules, test.expected.rules.rules) {
				t.Errorf("Expected %+v but got %+v", test.expected.rules.rules, cfg.rules.rules)
			}
		}()
	}
}

func TestLoadDir(t *testing.T) {
	filename := "test_config"

	dir, _ := ioutil.TempDir("", "temp_dir")
	file, _ := ioutil.TempFile(dir, filename)

	newFileName := strings.Join([]string{file.Name(), ".yml"}, "")
	os.Rename(file.Name(), newFileName)

	defer os.RemoveAll(dir)

	var data = `
      awx:
        address: https://my-awx.example.com/api
        proxy: http://my-proxy.example.com:3128
      rules:
      - metadata:
          name: start-node
        labels:
          alertname: "NodeDown"
        awxJob:
          template: "Start node"`

	expected := &Config{
		awx: &AWXConfig{
			address: "https://my-awx.example.com/api",
			proxy:   "http://my-proxy.example.com:3128",
			jobStatusCheckInterval: time.Duration(5) * time.Minute,
			ca: new(bytes.Buffer),
		},
		throttling: &ThrottlingConfig{
			interval: time.Duration(1) * time.Hour,
		},
		rules: &RulesConfig{
			rules: []*autoheal.HealingRule{
				{
					ObjectMeta: meta.ObjectMeta{
						Name: "start-node",
					},
					Labels: map[string]string{
						"alertname": "NodeDown",
					},
					AWXJob: &autoheal.AWXJobAction{
						Template: "Start node",
					},
				},
			},
		},
	}

	file.WriteString(data)

	l := NewBuilder()
	l.File(dir)

	cfg, err := l.Build()
	if err != nil {
		t.Errorf("An error occured! %s", err)
	}
	defer cfg.ShutDown()

	if !reflect.DeepEqual(cfg.awx, expected.awx) {
		t.Errorf("Expected %+v but got %+v", expected.awx, cfg.awx)
	}

	if !reflect.DeepEqual(cfg.throttling, expected.throttling) {
		t.Errorf("Expected %+v but got %+v", expected.throttling, cfg.throttling)
	}

	if !reflect.DeepEqual(cfg.rules.rules, expected.rules.rules) {
		t.Errorf("Expected %+v but got %+v", expected.rules.rules, cfg.rules.rules)
	}
}
