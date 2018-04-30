// Copyright 2018 Yaacov Zamir <kobi.zamir@gmail.com>
// and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package observer implements an event emitter and listener with builtin file watcher.
package observer

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"github.com/yaacov/observer/observer/set"
)

// WatchEvent (fsnotify.Event) represents a single file system notification.
type WatchEvent fsnotify.Event

// Listener is the function type to run on events.
type Listener func(interface{})

// Observer emplements the observer pattern.
type Observer struct {
	quit          chan bool
	events        chan interface{}
	watcher       *fsnotify.Watcher
	watchPatterns set.Set
	watchDirs     set.Set
	listeners     []Listener
	Verbose       bool
}

// Open the observer channles and run the event loop,
// it will return an error if event loop already running.
func (o *Observer) Open() error {
	if o.events != nil {
		return fmt.Errorf("Observer already inititated.")
	}

	// Create the observer channels.
	o.quit = make(chan bool)
	o.events = make(chan interface{})

	// Run the observer.
	return o.eventLoop()
}

// Close the observer channles,
// it will return an error if close fails.
func (o *Observer) Close() error {
	// Close event loop
	if o.events != nil {
		// Send a quit signal.
		o.quit <- true

		// Close channels.
		close(o.quit)
		close(o.events)
	}

	// Close file watcher.
	if o.watcher != nil {
		o.watcher.Close()
	}

	return nil
}

// AddListener adds a listener function to run on event,
// the listener function will recive the event object as argument.
// It will return an error if adding the new listener fails.
func (o *Observer) AddListener(l Listener) error {
	o.listeners = append(o.listeners, l)

	return nil
}

// Emit an event, and event can be of any type, when event is triggered all
// listeners will be called using the event object.
func (o *Observer) Emit(event interface{}) {
	o.events <- event
}

// Watch for file changes, watching a file can be done using exact file name,
// or shell pattern matching.
func (o *Observer) Watch(files []string) error {
	// Init watcher on first call.
	if o.watcher == nil {
		err := o.watchLoop()
		if err != nil {
			return err
		}
	}

	// Add file patterns and dirs to watch list.
	for _, f := range files {
		// For example if file is '/home/.config/*.conf':
		// base will be '*.conf'
		// dir will be '/home/.config'
		base := filepath.Base(f)
		dir := filepath.Dir(f)

		// Pattern calculation does not allways equal f from user.
		// We can not use the user provided file name here, because
		// in cases where we have no directory with the file name, we
		// do want to add the current directory './' before the base file
		// name. We can not use filepath.Join for the same reason, it will
		// remove the './' prefix when cleaning filename.
		pattern := fmt.Sprintf("%s%s%s", dir, string(filepath.Separator), base)

		// Logging file patterns
		if o.Verbose {
			log.Printf("[Debug] Adding pattern: %s", pattern)
		}
		o.watchPatterns.Add(pattern)
		o.watchDirs.Add(dir)
	}

	// NOTE: We watch directories and not files.
	//
	// We are watching directories and not files, because some text editors
	// and automated configuration systems may use clone-delete-rename pattern
	// instead of editing config files inline.
	// When a files is watched by name ane deleted, fsnotify will stop send
	// notifications for this file, watching a directory we will pick up
	// the new file with the same name and continue to get notifications.
	for _, d := range o.watchDirs.Values() {
		err := o.watcher.Add(d)
		if err != nil {
			return err
		}

		// Logging watched directories
		if o.Verbose {
			log.Printf("[Debug] Watching dir: %s", d)
		}
	}

	return nil
}

// handleEvent handle an event.
func (o *Observer) handleEvent(event interface{}) {
	// Run all listeners for this event.
	for _, listener := range o.listeners {
		go listener(event)
	}
}

// eventLoop runs the event loop.
func (o *Observer) eventLoop() error {
	// Run observer.
	go func() {
		for {
			select {
			case event := <-o.events:
				o.handleEvent(event)
			case <-o.quit:
				return
			}
		}
	}()

	return nil
}

// matchFile returns a boolean asserting whether this file is watched or not.
func (o Observer) matchFile(f string) (match bool) {
	// Look for an exact match.
	match = o.watchPatterns.Has(f)
	if match {
		return
	}

	// Try to match shell file name pattern.
	for _, p := range o.watchPatterns.Values() {
		match, _ = filepath.Match(p, f)
		if match {
			return
		}
	}

	return
}

// watchLoop runs a watcher loop for file changes.
func (o *Observer) watchLoop() error {
	var err error

	o.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Listen for file/directory changes.
	go func() {
		for {
			select {
			case event := <-o.watcher.Events:
				// Logging all events
				if o.Verbose {
					log.Printf("[Debug] Received event: %v", event)
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					// Check for event filename pattern match.
					if o.matchFile(event.Name) {
						o.handleEvent(WatchEvent(event))
					}
				}
			case err := <-o.watcher.Errors:
				if err != nil {
					o.handleEvent(err)
				}
			}
		}
	}()

	return nil
}
