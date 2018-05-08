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

// Package config contains types and functions used to load the service configuration.
//
package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/yaacov/observer/observer"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/internal/data"
)

// Config is a read only view of the configuration of the auto-heal service.
//
type Config struct {
	awx        *AWXConfig
	throttling *ThrottlingConfig
	rules      *RulesConfig
	listener   *eventListener

	// The names of the configuration files, in the order that they should be loaded:
	files     []string
	loadMutex *sync.Mutex
}

// AWX returns a read only view of the section of the configuration of the auto-heal service that
// describes how to connect to the AWX server, and how to launch jobs from templates.
//
func (c *Config) AWX() *AWXConfig {
	return c.awx
}

// Throttling returns a read only view of the section of the configuration that describes how to
// throttle the execution of healing rules.
//
func (c *Config) Throttling() *ThrottlingConfig {
	return c.throttling
}

// Rules returns the list of healing rules defined in the configuration.
//
func (c *Config) Rules() []*autoheal.HealingRule {
	return c.rules.rules
}

// ShutDown close the change obeserver channels
//
func (c *Config) ShutDown() {
	c.listener.shutDown()
}

// AddChangeListener to be called on config object update
//
func (c *Config) AddChangeListener(listener ChangeListener) {
	c.listener.addChangeListener(listener)
}

// watch config files for changes
//
func (c *Config) watch() {
	e := c.listener
	e.open()

	// Load config files
	c.load()

	// Start watching config files for modifications.
	configFiles := c.configFiles()
	err := e.configFilesChangedObserver.Watch(configFiles)
	if err != nil {
		glog.Errorf("Can't watch configuration files: %s", err)
		return
	}
	for _, file := range configFiles {
		glog.Infof("Watching configuration file '%s'", file)
	}

	// Load new configuration when config files change.
	e.configFilesChangedObserver.AddListener(func(_ interface{}) {
		// Reload the configuration files:
		glog.Infof("Configuration files have changed")
		err := c.load()
		if err != nil {
			glog.Errorf("Can't reload configuration files: %s", err)
			return
		}

		// If config files loaded succesfully emit config object changed event.
		e.configFilesLoadedObserver.Emit(observer.WatchEvent{Name: "Config loaded"})
	})
}

// load the configuration files and returns an error on fail.
//
func (c *Config) load() (err error) {
	// Loading the configuration modifies the members of the structure in place, so we need to avoid
	// running it simultaneously from multiple goroutines:
	c.loadMutex.Lock()
	defer c.loadMutex.Unlock()

	// Always clean rules before loading new ones
	c.rules.clear()

	// Merge the contents of the files into the empty configuration:
	for _, file := range c.files {
		var info os.FileInfo
		info, err = os.Stat(file)
		if err != nil {
			err = fmt.Errorf("Can't check if '%s' is a file or a directory: %s", file, err)
			return
		}
		if info.IsDir() {
			err = c.mergeDir(file)
			if err != nil {
				err = fmt.Errorf("Can't load configuration directory '%s': %s", file, err)
				return
			}
		} else {
			err = c.mergeFile(file)
			if err != nil {
				err = fmt.Errorf("Can't load configuration file '%s': %s", file, err)
				return
			}
		}
	}

	return
}

func (c *Config) mergeDir(dir string) error {
	// List the files in the directory:
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	files := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.IsDir() {
			name := info.Name()
			if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
				file := filepath.Join(dir, name)
				files = append(files, file)
			}
		}
	}

	// Load the files in alphabetical order:
	sort.Strings(files)
	for _, file := range files {
		err := c.mergeFile(file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) mergeFile(file string) error {
	var err error

	// Read the content of the file:
	glog.Infof("Loading configuration file '%s'", file)
	var content []byte
	content, err = ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// Parse the YAML inside the file:
	var decoded data.Config
	err = yaml.Unmarshal(content, &decoded)
	if err != nil {
		return err
	}

	// Merge the configuration data from the file with the existing configuration:
	if decoded.AWX != nil {
		err = c.awx.merge(decoded.AWX)
		if err != nil {
			return err
		}
	}
	if decoded.Throttling != nil {
		err = c.throttling.merge(decoded.Throttling)
		if err != nil {
			return err
		}
	}
	if decoded.Rules != nil {
		err = c.rules.merge(decoded.Rules)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) configFiles() (files []string) {
	// Merge the contents of the files into the empty configuration:
	for _, file := range c.files {
		info, err := os.Stat(file)
		if err != nil {
			// Pass
		}
		if info.IsDir() {
			files = append(files, filepath.Join(file, "*.yml"))
			files = append(files, filepath.Join(file, "*.yaml"))
		} else {
			files = append(files, file)
		}
	}

	return
}
