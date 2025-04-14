// Copyright 2024 The Godel Rescheduler Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hashid

import (
	"sync"
)

type HashID interface {
	Get(string) int
	Lookup(int) string
	Exist(string) bool
	Len() int
}

type hashIDImpl struct {
	data   map[string]int
	lookup map[int]string
	id     int
	mu     sync.RWMutex
}

func NewHashID() HashID {
	return &hashIDImpl{
		data:   make(map[string]int),
		lookup: make(map[int]string),
		id:     0,
	}
}

func (h *hashIDImpl) Get(key string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.data[key]
	if !ok {
		h.data[key] = h.id
		h.lookup[h.id] = key
		h.id++
	}
	return h.data[key]
}

func (h *hashIDImpl) Lookup(id int) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	key := h.lookup[id]
	return key
}

func (h *hashIDImpl) Exist(key string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.data[key]
	return ok
}

func (h *hashIDImpl) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.id
}
