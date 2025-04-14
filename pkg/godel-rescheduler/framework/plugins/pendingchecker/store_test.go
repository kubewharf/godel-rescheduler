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

package pendingchecker

import (
	"testing"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"

	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	v1 "k8s.io/api/core/v1"
	pointer "k8s.io/utils/ptr"
)

func TestCheckMovementItem(t *testing.T) {
	tests := []struct {
		name           string
		pendingPods    map[string]int
		args           *config.PendingCheckerArgs
		pod            *v1.Pod
		expectedResult policies.CheckResult
	}{
		{
			name: "args is nil",
			pendingPods: map[string]int{
				"default": 2,
				"":        3,
			},
			pod:            testing_helper.MakePod().NodeSelector(map[string]string{"": "default"}).Obj(),
			expectedResult: policies.Pass,
		},
		{
			name: "no corresponding threshold, no default threshold",
			pendingPods: map[string]int{
				"default": 2,
				"":        3,
			},
			args:           &config.PendingCheckerArgs{},
			pod:            testing_helper.MakePod().NodeSelector(map[string]string{"": "default"}).Obj(),
			expectedResult: policies.Pass,
		},
		{
			name: "no corresponding threshold, has default threshold, pass",
			pendingPods: map[string]int{
				"default": 2,
				"":        3,
			},
			args: &config.PendingCheckerArgs{
				DefaultPendingThreshold: pointer.To(2),
			},
			pod:            testing_helper.MakePod().NodeSelector(map[string]string{"": "default"}).Obj(),
			expectedResult: policies.Pass,
		},
		{
			name: "no corresponding threshold, has default threshold, fail",
			pendingPods: map[string]int{
				"default": 2,
				"":        3,
			},
			args: &config.PendingCheckerArgs{
				DefaultPendingThreshold: pointer.To(1),
			},
			pod:            testing_helper.MakePod().NodeSelector(map[string]string{"": "default"}).Obj(),
			expectedResult: policies.MovedInstanceError,
		},
		{
			name: "has corresponding threshold, no default threshold, pass",
			pendingPods: map[string]int{
				"default": 2,
				"":        3,
			},
			args: &config.PendingCheckerArgs{
				PendingThresholds: map[string]int{
					"default": 2,
				},
			},
			pod:            testing_helper.MakePod().NodeSelector(map[string]string{"": "default"}).Obj(),
			expectedResult: policies.Pass,
		},
		{
			name: "has corresponding threshold, no default threshold, fail",
			pendingPods: map[string]int{
				"default": 2,
				"":        3,
			},
			args: &config.PendingCheckerArgs{
				PendingThresholds: map[string]int{
					"default": 1,
				},
			},
			pod:            testing_helper.MakePod().NodeSelector(map[string]string{"": "default"}).Obj(),
			expectedResult: policies.MovedInstanceError,
		},
		{
			name: "has corresponding threshold, has default threshold, pass",
			pendingPods: map[string]int{
				"default": 2,
				"":        3,
			},
			args: &config.PendingCheckerArgs{
				PendingThresholds: map[string]int{
					"default": 2,
				},
				DefaultPendingThreshold: pointer.To(3),
			},
			pod:            testing_helper.MakePod().NodeSelector(map[string]string{"": "default"}).Obj(),
			expectedResult: policies.Pass,
		},
		{
			name: "has corresponding threshold, has default threshold, fail",
			pendingPods: map[string]int{
				"default": 2,
				"":        3,
			},
			args: &config.PendingCheckerArgs{
				PendingThresholds: map[string]int{
					"default": 1,
				},
				DefaultPendingThreshold: pointer.To(3),
			},
			pod:            testing_helper.MakePod().NodeSelector(map[string]string{"": "default"}).Obj(),
			expectedResult: policies.MovedInstanceError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newPendingCheckerStore(tt.pendingPods, tt.args)
			gotRes := store.CheckMovementItem("", "", tt.pod)
			if tt.expectedResult != gotRes {
				t.Errorf("expected %s but got %s", tt.expectedResult, gotRes)
			}
		})
	}
}
