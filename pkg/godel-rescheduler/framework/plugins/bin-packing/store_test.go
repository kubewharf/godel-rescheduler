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

package binpacking

import (
	"testing"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"

	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	v1 "k8s.io/api/core/v1"
)

func TestCheckMovement_BinPacking(t *testing.T) {
	tests := []struct {
		name           string
		nodes          []*v1.Node
		pods           []*v1.Pod
		pod            *v1.Pod
		previousNode   string
		targetNode     string
		expectedResult policies.CheckResult
	}{
		{
			name: "reschedule successfully from lower cpu utilization node to higher cpu utilization node",
			nodes: []*v1.Node{
				testing_helper.MakeNode().Name("n1").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
				testing_helper.MakeNode().Name("n2").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
			},
			pods: []*v1.Pod{
				testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node("n1").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node("n2").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node("n2").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
			},
			pod: testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node("n2").
				Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
			previousNode:   "n2",
			targetNode:     "n1",
			expectedResult: policies.Pass,
		},
		{
			name: "reschedule failed from higher cpu utilization node to lower cpu utilization node",
			nodes: []*v1.Node{
				testing_helper.MakeNode().Name("n1").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
				testing_helper.MakeNode().Name("n2").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
			},
			pods: []*v1.Pod{
				testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node("n1").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node("n2").
					Req(map[v1.ResourceName]string{"cpu": "3"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node("n2").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
			},
			pod: testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node("n2").
				Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
			previousNode:   "n2",
			targetNode:     "n1",
			expectedResult: policies.MovementItemError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rCache := cache.New(commoncache.MakeCacheHandlerWrapper().Obj())
			for _, p := range tt.pods {
				rCache.AddPod(p)
			}
			for _, n := range tt.nodes {
				rCache.AddNode(n)
			}
			snapshot := cache.NewEmptySnapshot(commoncache.MakeCacheHandlerWrapper().Obj())
			rCache.UpdateSnapshot(snapshot)
			store := newBinPackingStore(snapshot, []config.ResourceItem{
				{
					Resource: "cpu",
					Weight:   1,
				},
			})
			bpStore := store.(*binPackingStore)
			gotRes := bpStore.CheckMovementItem(tt.previousNode, tt.targetNode, tt.pod)
			if tt.expectedResult != gotRes {
				t.Errorf("expected: %s but got: %s", tt.expectedResult, gotRes)
			}
		})
	}
}
