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

type depthChartImpl struct {
	chart    map[int]NodeSet
	hash     map[Node]int
	maxDepth int
}

func NewDepthChart() DepthChart {
	return &depthChartImpl{
		chart:    make(map[int]NodeSet),
		hash:     make(map[Node]int),
		maxDepth: 0,
	}
}

func (dc *depthChartImpl) AddNode(node Node, depth int) {
	if _, ok := dc.hash[node]; ok {
		return
	}
	if dc.chart[depth] == nil {
		dc.chart[depth] = NewSet()
	}
	dc.chart[depth].Insert(node)
	if depth > dc.maxDepth {
		dc.maxDepth = depth
	}
	dc.hash[node] = depth
}

func (dc *depthChartImpl) DeleteNode(node Node) {
	if _, ok := dc.hash[node]; !ok {
		return
	}
	depth := dc.hash[node]
	if dc.chart[depth] == nil {
		return
	}
	dc.chart[depth].Delete(node)
	if dc.chart[depth].Len() == 0 {
		delete(dc.chart, depth)
		if depth == dc.maxDepth {
			dc.maxDepth--
		}
	}
	delete(dc.hash, node)
}

func (dc *depthChartImpl) GetNodes(depth int) NodeSet {
	return dc.chart[depth]
}

func (dc *depthChartImpl) GetMaxDepth() int {
	return dc.maxDepth
}

func (dc *depthChartImpl) Len() int {
	return int(len(dc.hash))
}
