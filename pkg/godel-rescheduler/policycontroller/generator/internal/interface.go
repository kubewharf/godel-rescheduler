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

type GraphObj interface {
	GetName() string
	GetID() int
}

type Graph interface {
	AddEdge(a, b, c int)
	DeleteEdge(ids ...int)
	NumNode() int
	NumEdge() int

	Head(int) int
	Edge(idx int) int
	Next(idx int) int

	GetNode(int) Node
	GetEdge(int) Edge

	GetDin(int) int
}

type Edge interface {
	GraphObj
	GetFrom() int
	GetTo() int
}

type Node interface {
	GraphObj
}

type SCC interface {
	GetID(x int) int
	GetSccCnt() int
}

type SccEdge interface {
	Edge
	GetEdge() Edge
}

type SccNode interface {
	Node
	GetNodes() []Node
	AddNodes(...Node)
	RemoveNodes(...Node)
	NumNodes() int
	GetEdges() []Edge
	AddEdges(...Edge)
	RemoveEdges(...Edge)
	NumEdges() int
}

type DepthChart interface {
	AddNode(node Node, depth int)
	DeleteNode(node Node)
	GetNodes(depth int) NodeSet
	GetMaxDepth() int
	Len() int
}

type Set interface {
	Exist(item interface{}) bool
	Insert(item ...interface{})
	Delete(item ...interface{})
	Len() int
	List() []interface{}
}

type (
	NodeSet   Set
	EdgeSet   Set
	CheckFunc func(Graph, EdgeSet, ...Edge) error
)
