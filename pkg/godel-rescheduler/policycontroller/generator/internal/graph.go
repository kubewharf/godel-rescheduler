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

import "sync"

type graphImpl struct {
	nodeSource                      []Node
	edgeSource                      []Edge
	head, edgeto, weight, next, din []int
	from, pre                       []int
	n, m, tot, deleted              int
	mu                              sync.RWMutex
}

var _ Graph = &graphImpl{}

func NewGraph(nodeSource []Node, edgeSource []Edge) Graph {
	n, m := len(nodeSource), len(edgeSource)
	g := &graphImpl{
		nodeSource: nodeSource,
		edgeSource: edgeSource,
		head:       make([]int, n),
		edgeto:     make([]int, m),
		weight:     make([]int, m),
		next:       make([]int, m),
		din:        make([]int, n),
		from:       make([]int, m),
		pre:        make([]int, m),
		n:          n,
		m:          m,
		tot:        0,
		deleted:    0,
	}
	for i := range g.head {
		g.head[i] = -1
	}
	for i := range g.edgeto {
		g.pre[i] = -1
	}
	for i := range edgeSource {
		e := edgeSource[i]
		g.AddEdge(e.GetFrom(), e.GetTo(), e.GetID())
	}
	return g
}

func (g *graphImpl) AddEdge(a, b, c int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.head[a] >= 0 {
		g.pre[g.head[a]] = g.tot
	}
	g.from[g.tot] = a
	g.edgeto[g.tot], g.weight[g.tot], g.next[g.tot], g.head[a] = b, c, g.head[a], g.tot
	g.din[b]++
	g.tot++
}

func (g *graphImpl) DeleteEdge(ids ...int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, id := range ids {
		if id < 0 || id >= g.tot {
			continue
		}
		if g.pre[id] >= 0 {
			g.next[g.pre[id]] = g.next[id]
		} else {
			// This means the current id is head.
			g.head[g.from[id]] = g.next[id]
		}
		if g.next[id] >= 0 {
			g.pre[g.next[id]] = g.pre[id]
		}
		g.deleted++
	}
}

func (g *graphImpl) NumNode() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.n
}

func (g *graphImpl) NumEdge() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.tot - g.deleted
}

func (g *graphImpl) Head(u int) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.head[u]
}

func (g *graphImpl) Edge(idx int) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.edgeto[idx]
}

func (g *graphImpl) Weight(idx int) int {
	return g.weight[idx]
}

func (g *graphImpl) Next(idx int) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.next[idx]
}

func (g *graphImpl) GetNode(idx int) Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodeSource[idx]
}

func (g *graphImpl) GetEdge(idx int) Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.edgeSource[idx]
}

func (g *graphImpl) GetDin(idx int) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.din[idx]
}
