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

import (
	"fmt"
)

type sccnodeImpl struct {
	name string
	id   int
	// ...
	selfCircle EdgeSet
	nodeSet    NodeSet
}

var (
	_ Node    = &sccnodeImpl{}
	_ SccNode = &sccnodeImpl{}
)

func NewSccNode() SccNode {
	return &sccnodeImpl{
		selfCircle: NewSet(),
		nodeSet:    NewSet(),
	}
}

func (n *sccnodeImpl) GetName() string {
	return n.name
}

func (n *sccnodeImpl) SetName(name string) {
	n.name = name
}

func (n *sccnodeImpl) GetID() int {
	return n.id
}

func (n *sccnodeImpl) SetID(id int) {
	n.id = id
}

func (n *sccnodeImpl) String() string {
	var nodes string
	for _, obj := range n.nodeSet.List() {
		node := obj.(Node)
		nodes += fmt.Sprintf("{%v}", node.GetName())
	}
	return fmt.Sprintf("{Name:%v{%v},ID:%v}", n.name, nodes, n.id)
}

func (n *sccnodeImpl) AddEdges(edges ...Edge) {
	for i := range edges {
		n.selfCircle.Insert(edges[i])
	}
}

func (n *sccnodeImpl) GetEdges() []Edge {
	objs := n.selfCircle.List()
	ret := make([]Edge, len(objs))
	for i := range objs {
		ret[i] = objs[i].(Edge)
	}
	return ret
}

func (n *sccnodeImpl) RemoveEdges(edges ...Edge) {
	for i := range edges {
		n.selfCircle.Delete(edges[i])
	}
}

func (n *sccnodeImpl) NumEdges() int {
	return int(n.selfCircle.Len())
}

func (n *sccnodeImpl) AddNodes(nodes ...Node) {
	for i := range nodes {
		n.nodeSet.Insert(nodes[i])
	}
}

func (n *sccnodeImpl) GetNodes() []Node {
	objs := n.nodeSet.List()
	ret := make([]Node, len(objs))
	for i := range objs {
		ret[i] = objs[i].(Node)
	}
	return ret
}

func (n *sccnodeImpl) RemoveNodes(nodes ...Node) {
	for i := range nodes {
		n.nodeSet.Delete(nodes[i])
	}
}

func (n *sccnodeImpl) NumNodes() int {
	return int(n.nodeSet.Len())
}
