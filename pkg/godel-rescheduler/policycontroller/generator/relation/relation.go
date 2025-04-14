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

package relation

import "fmt"

type RelationAttr interface{}

type Relation interface {
	GetFrom() string
	SetFrom(string)
	GetTo() string
	SetTo(string)
	GetName() string
	SetName(string)
	String() string
}

type Relations []Relation

func (s Relations) String() string {
	var str string
	for i := range s {
		str += fmt.Sprintf("%v,", s[i])
	}
	return fmt.Sprintf("[%v]", str)
}

type relationImpl struct {
	from, to string
	name     string
	// attribute RelationAttr
}

var (
	_ Relation = &relationImpl{}
)

func (r *relationImpl) GetFrom() string {
	return r.from
}

func (r *relationImpl) SetFrom(from string) {
	r.from = from
}

func (r *relationImpl) GetTo() string {
	return r.to
}

func (r *relationImpl) SetTo(to string) {
	r.to = to
}

func (r *relationImpl) GetName() string {
	return r.name
}

func (r *relationImpl) SetName(name string) {
	r.name = name
}

func (r *relationImpl) String() string {
	return fmt.Sprintf("{Name:%s,From:%s,To:%s}", r.name, r.from, r.to)
}

type RelationWrapper struct{ obj *relationImpl }

func MakeRelation() *RelationWrapper {
	return &RelationWrapper{&relationImpl{}}
}

func (w *RelationWrapper) Obj() Relation {
	return w.obj
}

func (w *RelationWrapper) Name(name string) *RelationWrapper {
	w.obj.name = name
	return w
}

func (w *RelationWrapper) From(from string) *RelationWrapper {
	w.obj.from = from
	return w
}

func (w *RelationWrapper) To(to string) *RelationWrapper {
	w.obj.to = to
	return w
}
