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
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	reschedulertesting "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/testing"

	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	godel_framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

func TestComputeResourceUtilization(t *testing.T) {
	n := testing_helper.MakeNode().Capacity(map[v1.ResourceName]string{"cpu": "10", "memory": "10"}).Obj()
	nInfo := godel_framework.NewNodeInfo()
	nInfo.SetNode(n)
	pods := []*v1.Pod{
		testing_helper.MakePod().UID("p1").Req(map[v1.ResourceName]string{"cpu": "1", "memory": "6"}).Obj(),
		testing_helper.MakePod().UID("p2").Req(map[v1.ResourceName]string{"cpu": "2", "memory": "2"}).Obj(),
	}
	for _, p := range pods {
		nInfo.AddPod(p)
	}
	thresholds := []config.ResourceItem{
		{
			Resource: "cpu",
			Weight:   1,
		},
		{
			Resource: "memory",
			Weight:   2,
		},
	}
	gotResult := computeResourceUtilization(nInfo, thresholds)
	if gotResult != 0.6333333333333334 {
		t.Errorf("unexpected result: %v", gotResult)
	}
}

func TestReschedule_BinPacking(t *testing.T) {
	tests := []struct {
		name           string
		nodes          []*v1.Node
		pods           []*v1.Pod
		expectedResult api.RescheduleResult
	}{
		{
			name: "rescheduling failed",
			nodes: []*v1.Node{
				testing_helper.MakeNode().Name("n1").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
				testing_helper.MakeNode().Name("n2").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
				testing_helper.MakeNode().Name("n3").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
			},
			pods: []*v1.Pod{
				testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node("n1").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node("n1").
					Req(map[v1.ResourceName]string{"cpu": "4"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node("n2").
					Req(map[v1.ResourceName]string{"cpu": "4"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p4").UID("p4").Node("n2").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p5").UID("p5").Node("n3").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p6").UID("p6").Node("n3").
					Req(map[v1.ResourceName]string{"cpu": "4"}).Obj(),
			},
			expectedResult: api.RescheduleResult{},
		},
		{
			name: "rescheduling sucessfully from lower utilization node to higher utilization node",
			nodes: []*v1.Node{
				testing_helper.MakeNode().Name("n1").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
				testing_helper.MakeNode().Name("n2").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
				testing_helper.MakeNode().Name("n3").Capacity(map[v1.ResourceName]string{"cpu": "10"}).Obj(),
			},
			pods: []*v1.Pod{
				testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node("n1").
					Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node("n2").
					Req(map[v1.ResourceName]string{"cpu": "3"}).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node("n3").
					Req(map[v1.ResourceName]string{"cpu": "4"}).Obj(),
			},
			expectedResult: api.RescheduleResult{
				PodMovements: []api.PodMovement{
					{
						PreviousNode: "n1",
						TargetNode:   "n3",
						Pod: testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node("n1").
							Req(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
					},
					{
						PreviousNode: "n2",
						TargetNode:   "n3",
						Pod: testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node("n2").
							Req(map[v1.ResourceName]string{"cpu": "3"}).Obj(),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rCache := cache.New(commoncache.MakeCacheHandlerWrapper().Obj())
			for _, n := range tt.nodes {
				rCache.AddNode(n)
			}
			for _, p := range tt.pods {
				if err := rCache.AddPod(p); err != nil {
					klog.Errorf("add pod %s failed: %v", p.UID, err)
				}
			}
			snapshot := cache.NewEmptySnapshot(commoncache.MakeCacheHandlerWrapper().Obj())
			rCache.UpdateSnapshot(snapshot)

			findNodesThatFitPod := func(ctx context.Context, pod *v1.Pod, nodes []godel_framework.NodeInfo) ([]godel_framework.NodeInfo, godel_framework.NodeToStatusMap, error) {
				return snapshot.List(), nil, nil
			}
			rfh := reschedulertesting.NewMockReschedulerFrameworkHandle(findNodesThatFitPod, rCache, snapshot, nil)
			plugin := &BinPacking{
				rfh: rfh,
				Args: &config.BinPackingArgs{
					ResourceItems: []config.ResourceItem{
						{
							Resource: "cpu",
							Weight:   1,
						},
					},
					ThresholdPercentage: 0.5,
				},
			}
			detectingResult, err := plugin.Detect(context.Background())
			if err != nil {
				t.Errorf("unexpected detecting error: %v", err)
			}
			reschedulingResult, err := plugin.Reschedule(context.Background(), detectingResult)
			if err != nil {
				t.Errorf("unexpected rescheduling error: %v", err)
			}
			sort.SliceStable(reschedulingResult.PodMovements, func(i, j int) bool {
				return reschedulingResult.PodMovements[i].Pod.UID < reschedulingResult.PodMovements[j].Pod.UID
			})
			if !reflect.DeepEqual(tt.expectedResult, reschedulingResult) {
				t.Errorf("expected %v but got %v", tt.expectedResult, reschedulingResult)
			}
		})
	}

}
