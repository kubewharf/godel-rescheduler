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

package throttle

import (
	"testing"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/api"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"

	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	"github.com/kubewharf/godel-scheduler/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pointer "k8s.io/utils/ptr"
)

func TestCheckMovementItem(t *testing.T) {
	assumedPods := []*v1.Pod{
		testing_helper.MakePod().Namespace("foo1").Name("foo1").UID("foo1").Label("psm", "a").Obj(),
		testing_helper.MakePod().Namespace("foo3").Name("foo3").UID("foo3").Label("psm", "c").Obj(),
		testing_helper.MakePod().Namespace("foo4").Name("foo4").UID("foo4").Label("psm", "c").Obj(),
	}

	tests := []struct {
		name        string
		pod         *v1.Pod
		expectedRes policies.CheckResult
	}{
		{
			name:        "step 1: pass for psm a",
			pod:         testing_helper.MakePod().Namespace("p1").Name("p1").UID("p1").Label("psm", "a").Obj(),
			expectedRes: policies.Pass,
		},
		{
			name:        "step 2: pass for psm b",
			pod:         testing_helper.MakePod().Namespace("p2").Name("p2").UID("p2").Label("psm", "b").Obj(),
			expectedRes: policies.Pass,
		},
		{
			name:        "step 3: fail for psm a",
			pod:         testing_helper.MakePod().Namespace("p3").Name("p3").UID("p3").Label("psm", "a").Obj(),
			expectedRes: policies.MovedInstanceError,
		},
		{
			name:        "step 4: pass for psm b",
			pod:         testing_helper.MakePod().Namespace("p4").Name("p4").UID("p4").Label("psm", "b").Obj(),
			expectedRes: policies.Pass,
		},
		{
			name:        "step 5: fail for psm b",
			pod:         testing_helper.MakePod().Namespace("p5").Name("p5").UID("p5").Label("psm", "b").Obj(),
			expectedRes: policies.MovedInstanceError,
		},
		{
			name:        "step 6: fail for psm c",
			pod:         testing_helper.MakePod().Namespace("p6").Name("p6").UID("p6").Label("psm", "c").Obj(),
			expectedRes: policies.MovedInstanceError,
		},
	}

	rCache := cache.New(commoncache.MakeCacheHandlerWrapper().Obj())
	for _, p := range assumedPods {
		pInfo := framework.MakeCachePodInfoWrapper().Pod(p).Obj()
		reschedPodInfo := api.MakeCachePodInfoWrapper().PodInfo(pInfo).Obj()
		rCache.AssumePod(reschedPodInfo)
	}
	snapshot := cache.NewEmptySnapshot(commoncache.MakeCacheHandlerWrapper().Obj())
	rCache.UpdateSnapshot(snapshot)
	throttleItems := []config.ThrottleItem{
		{
			LabelSelectorKey: "psm",
			ThrottleDuration: metav1.Duration{Duration: time.Hour},
			ThrottleValue:    2,
		},
	}
	store := newThrottleStore(snapshot, throttleItems)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes := store.CheckMovementItem("", "", tt.pod)
			if tt.expectedRes != gotRes {
				t.Errorf("expected: %s but got: %s", tt.expectedRes, gotRes)
			}
			if tt.expectedRes == policies.Pass {
				store.AssumePod(tt.pod)
			}
		})
	}
}

func TestCheckMovementItem_MultipleConfigItems(t *testing.T) {
	assumedPods := []*v1.Pod{
		testing_helper.MakePod().Namespace("foo1").Name("foo1").UID("foo1").Label("name", "a").Obj(),
		testing_helper.MakePod().Namespace("foo2").Name("foo2").UID("foo2").Label("name", "a").Obj(),
		testing_helper.MakePod().Namespace("foo3").Name("foo3").UID("foo3").Label("name", "a").Obj(),
	}

	tests := []struct {
		name          string
		throttleItems []config.ThrottleItem
		pod           *v1.Pod
		expectedRes   policies.CheckResult
	}{
		{
			name: "pass for deploy a, a can not be preempted, multiple throttle items",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "name",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    4,
					CanBePreempted:   pointer.To(false),
				},
				{
					LabelSelectorKey: "name",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    3,
					CanBePreempted:   pointer.To(true),
				},
			},
			pod:         testing_helper.MakePod().Namespace("p").Name("p").UID("p").Label("name", "a").Obj(),
			expectedRes: policies.Pass,
		},
		{
			name: "fail for deploy a, a can be preempted, multiple throttle items",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "name",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    4,
					CanBePreempted:   pointer.To(false),
				},
				{
					LabelSelectorKey: "name",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    3,
					CanBePreempted:   pointer.To(true),
				},
			},
			pod: testing_helper.MakePod().Namespace("p").Name("p").UID("p").
				PriorityClassName("pc").
				Label("name", "a").
				Annotation(util.CanBePreemptedAnnotationKey, util.CanBePreempted).Obj(),
			expectedRes: policies.MovedInstanceError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rCache := cache.New(commoncache.MakeCacheHandlerWrapper().Obj())
			for _, p := range assumedPods {
				pInfo := framework.MakeCachePodInfoWrapper().Pod(p).Obj()
				reschedPodInfo := api.MakeCachePodInfoWrapper().PodInfo(pInfo).Obj()
				rCache.AssumePod(reschedPodInfo)
			}
			snapshot := cache.NewEmptySnapshot(commoncache.MakeCacheHandlerWrapper().Obj())
			rCache.UpdateSnapshot(snapshot)
			store := newThrottleStore(snapshot, tt.throttleItems).(*throttleStore)

			gotRes := store.CheckMovementItem("", "", tt.pod)
			if tt.expectedRes != gotRes {
				t.Errorf("expected: %s but got: %s", tt.expectedRes, gotRes)
			}
		})
	}
}
