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

package internal

import "fmt"

type setImpl struct {
	data map[interface{}]struct{}
}

func NewSet() Set {
	return &setImpl{
		data: make(map[interface{}]struct{}),
	}
}

func (s *setImpl) Exist(item interface{}) bool {
	if s == nil {
		return false
	}
	_, ok := s.data[item]
	return ok
}

func (s *setImpl) Insert(items ...interface{}) {
	if s == nil {
		return
	}
	for _, item := range items {
		if _, ok := s.data[item]; ok {
			continue
		}
		s.data[item] = struct{}{}
	}
}

func (s *setImpl) Delete(items ...interface{}) {
	if s == nil {
		return
	}
	for _, item := range items {
		if _, ok := s.data[item]; !ok {
			continue
		}
		delete(s.data, item)
	}
}

func (s *setImpl) Len() int {
	if s == nil {
		return 0
	}
	return len(s.data)
}

func (s *setImpl) List() []interface{} {
	if s == nil {
		return nil
	}
	ret := make([]interface{}, 0)
	for k := range s.data {
		ret = append(ret, k)
	}
	return ret
}

func (s *setImpl) String() string {
	if s == nil {
		return "{}"
	}
	var str string
	for k := range s.data {
		str += fmt.Sprintf("{%v},", s.data[k])
	}
	return fmt.Sprintf("{%v}", str)
}
