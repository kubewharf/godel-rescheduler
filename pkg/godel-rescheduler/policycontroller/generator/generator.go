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

package generator

import (
	"context"
	"fmt"
	"sync"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/internal"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/relation"
	"github.com/kubewharf/godel-rescheduler/pkg/util/hashid"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
)

type Generator interface {
	Finished() bool
	Generate(context.Context) (relation.Relations, error)
}

type generatorImpl struct {
	relations        relation.Relations
	nodesID, edgesID hashid.HashID
	reverseG         internal.Graph
	chart            internal.DepthChart
	checkFunc        internal.CheckFunc
	mu               sync.RWMutex
}

func NewGenerator(ctx context.Context, relations relation.Relations, checkFunc internal.CheckFunc) Generator {
	klog.V(4).InfoS("NewGenerator", "relations", relations)
	nodes, edges := make([]internal.Node, 0), make([]internal.Edge, 0)
	nodesID, edgesID := hashid.NewHashID(), hashid.NewHashID()
	{
		for _, relation := range relations {
			nodeFromName, nodeToName, podName := relation.GetFrom(), relation.GetTo(), relation.GetName()
			// TODO(libing.binacs@):
			// 1. In order to support rescheduling at the single-node micro-topology level, we temporarily
			//   comment this part of the code judgment.
			// 2. In the long run, we should configure it according to different strategy requirements to avoid
			//   unnecessary rescheduling behavior.
			// if nodeFromName == nodeToName {
			// 	continue
			// }
			klog.V(4).InfoS("Generator parse relation", "from", nodeFromName, "to", nodeToName, "podName", podName)
			existFrom, existTo := nodesID.Exist(nodeFromName), nodesID.Exist(nodeToName)
			nodeFromID, nodeToID, podID := nodesID.Get(nodeFromName), nodesID.Get(nodeToName), edgesID.Get(podName)
			if !existFrom {
				node := internal.MakeNode().Name(nodeFromName).ID(nodeFromID).Obj()
				nodes = append(nodes, node)
				klog.V(4).InfoS("Generator add node", "node", node)
			}
			if !existTo {
				node := internal.MakeNode().Name(nodeToName).ID(nodeToID).Obj()
				nodes = append(nodes, node)
				klog.V(4).InfoS("Generator add node", "node", node)
			}
			edge := internal.MakeEdge().Name(podName).ID(podID).From(nodeFromID).To(nodeToID).Obj()
			edges = append(edges, edge)
			klog.V(4).InfoS("Generator add edge", "edge", edge)
		}

	}
	klog.V(4).InfoS("NewGenerator", "nodes", nodes, "edges", edges)

	reverseG, chart := Process(ctx, nodes, edges)
	klog.V(4).InfoS("NewGenerator", "reverseG", reverseG, "chart", chart)
	return &generatorImpl{
		relations: relations,
		nodesID:   nodesID,
		edgesID:   edgesID,
		reverseG:  reverseG,
		chart:     chart,
		checkFunc: checkFunc,
	}
}

func (s *generatorImpl) Finished() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.chart.Len() == 0
}

func (s *generatorImpl) Generate(ctx context.Context) (relation.Relations, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	es, err := Select(ctx, s.reverseG, s.chart, s.checkFunc)
	if err != nil {
		if s.chart.Len() > 0 {
			// TODO: retry and timeout
			return nil, fmt.Errorf("invalid graph, cann't pass the CheckFunc, error message: %v", err)
		}
		return nil, nil
	}
	relations := make(relation.Relations, 0)
	for _, obj := range es.List() {
		edge := obj.(internal.Edge)
		nodeFrom, nodeTo, podName := s.nodesID.Lookup(edge.GetFrom()), s.nodesID.Lookup(edge.GetTo()), s.edgesID.Lookup(edge.GetID())
		relations = append(relations, relation.MakeRelation().From(nodeFrom).To(nodeTo).Name(podName).Obj())
	}
	klog.InfoS("Generate", "SelectRelations", relations)
	return relations, nil
}

func Process(ctx context.Context, nodes []internal.Node, edges []internal.Edge) (internal.Graph, internal.DepthChart) {
	g := internal.NewGraph(nodes, edges)
	scc := internal.SCCTarjan(g)
	count := scc.GetSccCnt()

	sccNodes, sccEdges := make([]internal.Node, count), make([]internal.Edge, 0)
	{
		for i := 0; i < count; i++ {
			// Make a SccNode instead of a Node
			name := fmt.Sprintf("SccSccNode-%v", i)
			sccNodes[i] = internal.MakeSccNode().Name(name).ID(i).Obj()
		}
		for i := 0; i < g.NumNode(); i++ {
			sccNodes[scc.GetID(i)].(internal.SccNode).AddNodes(g.GetNode(i))
			for j := g.Head(i); j != -1; j = g.Next(j) {
				k := g.Edge(j)
				a, b := scc.GetID(i), scc.GetID(k)
				if a == b {
					sccNodes[scc.GetID(i)].(internal.SccNode).AddEdges(g.GetEdge(j))
				} else {
					// ATTENTION: make a reverse graph (b->a).
					nodeEdge := g.GetEdge(j)
					name := fmt.Sprintf("SccEdge-%v", j)
					sccEdge := internal.MakeSccEdge().Name(name).ID(j).From(b).To(a).Edge(nodeEdge).Obj()
					sccEdges = append(sccEdges, sccEdge)
				}
			}
		}
		// TODO: Check nodeSet and edges in advance if we can.
	}

	reverseG := internal.NewGraph(sccNodes, sccEdges)
	depthChart := internal.NewDepthChart()
	{
		depth := make([]int, count)
		for i := count - 1; i >= 0; i-- {
			for j := reverseG.Head(i); j != -1; j = reverseG.Next(j) {
				k := reverseG.Edge(j)
				if depth[i] < depth[k]+1 {
					depth[i] = depth[k] + 1
				}
			}
		}
		for i := count - 1; i >= 0; i-- {
			depthChart.AddNode(reverseG.GetNode(i), depth[i])
		}
	}

	// Now we got a DAG and a deep chart.
	return reverseG, depthChart
}

func Select(ctx context.Context, reverseG internal.Graph, chart internal.DepthChart, checkFunc internal.CheckFunc) (internal.EdgeSet, error) {
	var (
		errInSelfEdges    error
		errInNonSelfEdges error
	)
	selectedEdges := internal.NewSet()

	// TODO: improve the following strategy.
	for dep := chart.GetMaxDepth(); dep >= 0; dep-- {
		sccNodeSet, changed := chart.GetNodes(dep), false

		// 1. select self edges
		for _, obj := range sccNodeSet.List() {
			sccNode := obj.(internal.SccNode)
			selfEdges := sccNode.GetEdges()
			// TODO: improve check function.
			if err := checkFunc(reverseG, selectedEdges, selfEdges...); err == nil {
				for i := range selfEdges {
					selectedEdges.Insert(selfEdges[i])
				}
				sccNode.RemoveEdges(selfEdges...)
				changed = true
			} else {
				errInSelfEdges = fmt.Errorf("check failed for self circle: %v, previous selected edges: %v, error: %v", selfEdges, selectedEdges, err)
			}
		}

		// 2. select non-self edges
		for _, obj := range sccNodeSet.List() {
			sccNode := obj.(internal.SccNode)
			if sccNode.NumEdges() > 0 {
				continue
			}
			deleteSet := internal.NewSet()
			i := sccNode.GetID()
			for j := reverseG.Head(i); j != -1; j = reverseG.Next(j) {
				e := reverseG.GetEdge(j).(internal.SccEdge).GetEdge()
				if err := checkFunc(reverseG, selectedEdges, e); err == nil {
					selectedEdges.Insert(e)
					deleteSet.Insert(j)
					changed = true
				} else {
					errInNonSelfEdges = fmt.Errorf("check failed for edge: %v, previous selected edges: %v, error: %v", e, selectedEdges, err)
				}
			}
			if deleteSet.Len() > 0 {
				for _, obj := range deleteSet.List() {
					i := obj.(int)
					reverseG.DeleteEdge(i)
				}
			}
		}
		for _, obj := range sccNodeSet.List() {
			sccNode := obj.(internal.SccNode)
			if sccNode.NumEdges() == 0 && reverseG.Head(sccNode.GetID()) == -1 {
				chart.DeleteNode(sccNode)
			}
		}

		if !changed {
			break
		}

		if chart.GetNodes(dep) != nil && chart.GetNodes(dep).Len() > 0 {
			break
		}
	}

	if selectedEdges.Len() == 0 {
		// TODO: retry and wait for timeout.
		return nil, utilerrors.NewAggregate([]error{errInSelfEdges, errInNonSelfEdges})
	}

	return selectedEdges, nil
}
