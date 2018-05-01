/*
Copyright (c) 2018 Red Hat, Ine.

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
	"github.com/golang/glog"
	"github.com/yaacov/observer/observer"
)

// ChangeEvent contains config change event info
//
type ChangeEvent struct {
	// Empty
}

// ChangeListener a listener function that can be called when config event change is triggered.
//
type ChangeListener func(*ChangeEvent)

// eventListener is an event listener for autoheal configuration object
//
type eventListener struct {
	configFilesChangedObserver *observer.Observer
	configFilesLoadedObserver  *observer.Observer
}

// addChangeListener to be called on config object update
//
func (e *eventListener) addChangeListener(listener ChangeListener) {
	// add a new listener to configFilesChangedObserver
	e.configFilesLoadedObserver.AddListener(func(_ interface{}) {
		glog.Info("eventListener: Config object changed")
		listener(&ChangeEvent{})
	})
}

// shutDown close the change obeserver channels
//
func (e *eventListener) shutDown() {
	e.configFilesChangedObserver.Close()
	e.configFilesLoadedObserver.Close()
}

// open the change obeserver channels
//
func (e *eventListener) open() {
	// Start a change watcher over changed config files.
	if e.configFilesChangedObserver == nil {
		e.configFilesChangedObserver = new(observer.Observer)
		e.configFilesChangedObserver.Open()
	}

	// Start a change watcher over loaded config files.
	if e.configFilesLoadedObserver == nil {
		e.configFilesLoadedObserver = new(observer.Observer)
		e.configFilesLoadedObserver.Open()
	}
}
