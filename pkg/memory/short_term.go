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

// This file contains the implementation of the memory where the healer stores the list of actions
// that have already been performed.

package memory

import (
	"reflect"
	"sync"
	"time"
)

// ShortTermMemoryBuilder builds of short term memory objects.
//
type ShortTermMemoryBuilder struct {
	// How long to remember actions.
	duration time.Duration
}

// ShortTermMemory stores a set of items for a given period of time.
//
type ShortTermMemory struct {
	// How long to remember actions.
	duration time.Duration

	// There will be a cell for each action stored, containing the action itself and the time it was
	// added to the memory.
	cells []*ShortTermCell

	// Mutex used to prevent simultaneous updates of the data structures.
	mutex *sync.Mutex
}

// ShortTermCell stores each individual action, and the time it was added to the memory.
//
type ShortTermCell struct {
	// The item stored in the cell.
	item interface{}

	// The time when the cell was created or updated.
	stamp time.Time
}

// NewShortTermMemoryBuilder creates a builder that can create short term memory objects.
//
func NewShortTermMemoryBuilder() *ShortTermMemoryBuilder {
	b := new(ShortTermMemoryBuilder)
	return b
}

// Duration sets how long objects in the memory will be remembered. The default is zero, which means
// that objects won't be remembered at all.
//
func (b *ShortTermMemoryBuilder) Duration(duration time.Duration) *ShortTermMemoryBuilder {
	b.duration = duration
	return b
}

// Build creates a new short term memory object with the configuration stored in the builder.
//
func (b *ShortTermMemoryBuilder) Build() (m *ShortTermMemory, err error) {
	m = new(ShortTermMemory)
	m.duration = b.duration
	m.cells = make([]*ShortTermCell, 0)
	m.mutex = &sync.Mutex{}
	return
}

// Add adds a new item to the memory.
//
func (m *ShortTermMemory) Add(item interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cell := m.findMatchingCell(item)
	if cell == nil {
		cell = new(ShortTermCell)
		cell.item = item
		m.cells = append(m.cells, cell)
	}
	cell.stamp = time.Now()
}

func (m *ShortTermMemory) Has(item interface{}) bool {

	m.mutex.Lock()
	defer m.mutex.Unlock()
	// Purge cells before checking.
	m.purgeExpiredCells()
	return m.findMatchingCell(item) != nil
}

// Len returns the number of items inside the memory.
//
func (m *ShortTermMemory) Len() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.purgeExpiredCells()
	return len(m.cells)
}

// purgeExpiredCells finds the aged cells and removes them.
//
func (m *ShortTermMemory) purgeExpiredCells() {
	// The cells in ShortTermMemory are monotonously increasing - from "older" to "younger"
	// thus, it suffices to check for until age < duration and then - break.
	now := time.Now()
	for idx, cell := range m.cells {
		age := now.Sub(cell.stamp)
		if age >= m.duration {
			// zeroing the value of the cell so that it wouldn't be refrenced by the underlying array
			// causing GO's grabage collector to collect the allocated memory.
			m.cells[idx] = nil
			m.cells = append(m.cells[:idx], m.cells[idx+1:]...)
		} else {
			break
		}
	}
}

// findMatchingCell tries to find the cell that contains the given item and returs a pointer to that
// cell or else nil if no such cell exists. Note that this method assumes that the mutex has already
// been acquired and that the expired cells have already been purged.
//
func (m *ShortTermMemory) findMatchingCell(item interface{}) *ShortTermCell {
	for _, cell := range m.cells {
		if reflect.DeepEqual(item, cell.item) {
			return cell
		}
	}
	return nil
}

func (m *ShortTermMemory) Duration() time.Duration {
	return m.duration
}

// Purge the expired cells from the short term memory cache.
//
func (m *ShortTermMemory) Clean() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.purgeExpiredCells()
}
