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

package pdbchecker

import (
	"testing"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	reschedulerpodutil "github.com/kubewharf/godel-rescheduler/pkg/util/pod"

	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	pdbstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/pdb_store"
	preemptionstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/preemption_store"
	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
)

func TestCheckMovementItemAfterAssume(t *testing.T) {
	p1 := testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node("n1").Label("name", "foo1").Obj()
	p2 := testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node("n1").Label("name", "foo2").Obj()
	p3 := testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node("n1").Label("name", "foo1").Obj()
	p4 := testing_helper.MakePod().Namespace("default").Name("p4").UID("p4").Node("n1").Label("name", "foo2").Obj()

	pdb1 := testing_helper.MakePdb().Namespace("default").Name("pdb1").Label("name", "foo1").DisruptionsAllowed(1).Obj()
	pdb2 := testing_helper.MakePdb().Namespace("default").Name("pdb2").Label("name", "foo2").DisruptionsAllowed(2).Obj()

	rCache := cache.New(commoncache.MakeCacheHandlerWrapper().EnableStore(string(pdbstore.Name), string(preemptionstore.Name)).Obj())
	rCache.AddPod(p1)
	rCache.AddPod(p2)
	rCache.AddPod(p3)
	rCache.AddPod(p4)
	rCache.AddPDB(pdb1)
	rCache.AddPDB(pdb2)

	snapshot := cache.NewEmptySnapshot(commoncache.MakeCacheHandlerWrapper().EnableStore(string(pdbstore.Name), string(preemptionstore.Name)).Obj())
	rCache.UpdateSnapshot(snapshot)
	store := &pdbCheckerStore{
		cache: generateCache(snapshot),
	}

	tests := []struct {
		name        string
		pod         *v1.Pod
		expectedRes policies.CheckResult
	}{
		{
			name:        "check p1",
			pod:         p1,
			expectedRes: policies.Pass,
		},
		{
			name:        "check p2",
			pod:         p2,
			expectedRes: policies.Pass,
		},
		{
			name:        "check p3",
			pod:         p3,
			expectedRes: policies.MovedInstanceError,
		},
		{
			name:        "check p4",
			pod:         p4,
			expectedRes: policies.Pass,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes := store.CheckMovementItem("n2", "n1", tt.pod)
			if tt.expectedRes != gotRes {
				t.Errorf("expected: %s, but got: %s", tt.expectedRes, gotRes)
			}
			if gotRes != policies.Pass {
				return
			}
			podClone := reschedulerpodutil.PrepareReschedulePod(tt.pod)
			podClone.Annotations[podutil.AssumedNodeAnnotationKey] = "n1"
			podClone.Spec.NodeName = "n1"
			store.AssumePod(podClone)
		})
	}
}
