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

package groupthrottle

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
		testing_helper.MakePod().Namespace("foo1").Name("foo1").UID("foo1").Label("psm", "a").Label("name", "a").Obj(),
		testing_helper.MakePod().Namespace("foo2").Name("foo2").UID("foo2").Label("psm", "a").Label("name", "a").Obj(),
		testing_helper.MakePod().Namespace("foo3").Name("foo3").UID("foo3").Label("psm", "a").Label("name", "a").Obj(),
	}

	tests := []struct {
		name          string
		throttleItems []config.ThrottleItem
		indexes       []uint
		pod           *v1.Pod
		expectedRes   policies.CheckResult
	}{
		{
			name: "pass for psm a",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "psm",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    4,
				},
			},
			indexes:     []uint{0, 1, 2},
			pod:         testing_helper.MakePod().Namespace("p").Name("p").UID("p").Label("psm", "a").Obj(),
			expectedRes: policies.Pass,
		},
		{
			name: "pass for psm a, although pod count reach throttle threshold",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "psm",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    3,
				},
			},
			indexes:     []uint{0, 0, 1},
			pod:         testing_helper.MakePod().Namespace("p").Name("p").UID("p").Label("psm", "a").Obj(),
			expectedRes: policies.Pass,
		},
		{
			name: "fail for psm a",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "psm",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    3,
				},
			},
			indexes:     []uint{0, 1, 2},
			pod:         testing_helper.MakePod().Namespace("p").Name("p").UID("p").Label("psm", "a").Obj(),
			expectedRes: policies.MovedInstanceError,
		},
		{
			name: "fail for psm a",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "psm",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    2,
				},
			},
			indexes:     []uint{0, 1, 1},
			pod:         testing_helper.MakePod().Namespace("p").Name("p").UID("p").Label("psm", "a").Obj(),
			expectedRes: policies.MovedInstanceError,
		},
		{
			name: "fail for deploy b, b can be preempted, config can be preempted",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "name",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    3,
					CanBePreempted:   pointer.To(true),
				},
			},
			indexes: []uint{0, 1, 2},
			pod: testing_helper.MakePod().Namespace("p").Name("p").UID("p").
				PriorityClassName("pc").
				Label("name", "a").
				Annotation(util.CanBePreemptedAnnotationKey, util.CanBePreempted).Obj(),
			expectedRes: policies.MovedInstanceError,
		},
		{
			name: "fail for deploy b, b can not be preempted, config nil",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "name",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    3,
				},
			},
			indexes:     []uint{0, 1, 2},
			pod:         testing_helper.MakePod().Namespace("p").Name("p").UID("p").Label("name", "a").Obj(),
			expectedRes: policies.MovedInstanceError,
		},
		{
			name: "pass for deploy a, a can not be preempted, config can be preempted",
			throttleItems: []config.ThrottleItem{
				{
					LabelSelectorKey: "name",
					ThrottleDuration: metav1.Duration{Duration: time.Hour},
					ThrottleValue:    3,
					CanBePreempted:   pointer.To(true),
				},
			},
			indexes:     []uint{0, 1, 2},
			pod:         testing_helper.MakePod().Namespace("p").Name("p").UID("p").Label("name", "a").Obj(),
			expectedRes: policies.Pass,
		},
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
			indexes:     []uint{0, 1, 2},
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
			indexes: []uint{0, 1, 2},
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
			for i, p := range assumedPods {
				pInfo := framework.MakeCachePodInfoWrapper().Pod(p).Obj()
				reschedPodInfo := api.MakeCachePodInfoWrapper().PodInfo(pInfo).MovementGeneration(tt.indexes[i]).Obj()
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
