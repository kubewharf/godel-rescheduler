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

type NodeWrapper struct{ obj *nodeImpl }

func MakeNode() *NodeWrapper {
	return &NodeWrapper{&nodeImpl{}}
}

func (w *NodeWrapper) Obj() Node {
	return w.obj
}

func (w *NodeWrapper) Name(name string) *NodeWrapper {
	w.obj.name = name
	return w
}

func (w *NodeWrapper) ID(id int) *NodeWrapper {
	w.obj.id = id
	return w
}

type SccNodeWrapper struct{ obj *sccnodeImpl }

func MakeSccNode() *SccNodeWrapper {
	return &SccNodeWrapper{&sccnodeImpl{selfCircle: NewSet(), nodeSet: NewSet()}}
}

func (w *SccNodeWrapper) Obj() SccNode {
	return w.obj
}

func (w *SccNodeWrapper) Name(name string) *SccNodeWrapper {
	w.obj.name = name
	return w
}

func (w *SccNodeWrapper) ID(id int) *SccNodeWrapper {
	w.obj.id = id
	return w
}

func (w *SccNodeWrapper) Edges(selfCircle []Edge) *SccNodeWrapper {
	w.obj.selfCircle = NewSet()
	for i := range selfCircle {
		w.obj.selfCircle.Insert(selfCircle[i])
	}
	return w
}

func (w *SccNodeWrapper) Nodes(nodeSet []Node) *SccNodeWrapper {
	w.obj.nodeSet = NewSet()
	for i := range nodeSet {
		w.obj.nodeSet.Insert(nodeSet[i])
	}
	return w
}

type EdgeWrapper struct{ obj *edgeImpl }

func MakeEdge() *EdgeWrapper {
	return &EdgeWrapper{&edgeImpl{}}
}

func (w *EdgeWrapper) Obj() Edge {
	return w.obj
}

func (w *EdgeWrapper) Name(name string) *EdgeWrapper {
	w.obj.name = name
	return w
}

func (w *EdgeWrapper) ID(id int) *EdgeWrapper {
	w.obj.id = id
	return w
}

func (w *EdgeWrapper) From(from int) *EdgeWrapper {
	w.obj.from = from
	return w
}

func (w *EdgeWrapper) To(to int) *EdgeWrapper {
	w.obj.to = to
	return w
}

type SccEdgeWrapper struct{ obj *sccedgeImpl }

func MakeSccEdge() *SccEdgeWrapper {
	return &SccEdgeWrapper{&sccedgeImpl{}}
}

func (w *SccEdgeWrapper) Obj() SccEdge {
	return w.obj
}

func (w *SccEdgeWrapper) Name(name string) *SccEdgeWrapper {
	w.obj.name = name
	return w
}

func (w *SccEdgeWrapper) ID(id int) *SccEdgeWrapper {
	w.obj.id = id
	return w
}

func (w *SccEdgeWrapper) From(from int) *SccEdgeWrapper {
	w.obj.from = from
	return w
}

func (w *SccEdgeWrapper) To(to int) *SccEdgeWrapper {
	w.obj.to = to
	return w
}

func (w *SccEdgeWrapper) Edge(edge Edge) *SccEdgeWrapper {
	w.obj.edge = edge
	return w
}
