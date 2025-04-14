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

type edgeImpl struct {
	name string
	id   int
	// ...
	from, to int
}

var (
	_ Edge = &edgeImpl{}
)

func (e *edgeImpl) GetName() string {
	return e.name
}

func (e *edgeImpl) GetID() int {
	return e.id
}

func (e *edgeImpl) GetFrom() int {
	return e.from
}

func (e *edgeImpl) GetTo() int {
	return e.to
}

func (e *edgeImpl) String() string {
	return fmt.Sprintf("{Name:%v,ID:%v,From:%v,To:%v}", e.name, e.id, e.from, e.to)
}
