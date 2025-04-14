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

package pendingpodstore

import (
	"reflect"
	"testing"

	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
)

func TestUpdateSnapshot(t *testing.T) {
	cache := NewCache(nil)
	snapshot := NewSnapshot(nil)

	pa_1 := testing_helper.MakePod().Namespace("pa").Name("pa").UID("pa").Obj()
	pa_2 := testing_helper.MakePod().Namespace("pa").Name("pa").UID("pa").Node("n").Obj()
	pb_1 := testing_helper.MakePod().Namespace("pb").Name("pb").UID("pb").
		Annotation(podutil.PodStateAnnotationKey, string(podutil.PodAssumed)).
		Annotation(podutil.SchedulerAnnotationKey, "godel-scheduler-0").
		Annotation(podutil.AssumedNodeAnnotationKey, "n").Obj()
	pb_2 := testing_helper.MakePod().Namespace("pb").Name("pb").UID("pb").Obj()

	// add pending pod
	cache.AddPod(pa_1)
	cache.UpdateSnapshot(snapshot)
	gotPendingPods := snapshot.(*PendingPodStore).GetPendingPods()
	expectedPendingPods := map[string]int{
		"": 1,
	}
	if !reflect.DeepEqual(expectedPendingPods, gotPendingPods) {
		t.Errorf("expected %v but got %v", expectedPendingPods, gotPendingPods)
	}
	// update pending -> bound
	cache.UpdatePod(pa_1, pa_2)
	cache.UpdateSnapshot(snapshot)
	gotPendingPods = snapshot.(*PendingPodStore).GetPendingPods()
	expectedPendingPods = map[string]int{}
	if !reflect.DeepEqual(expectedPendingPods, gotPendingPods) {
		t.Errorf("expected %v but got %v", expectedPendingPods, gotPendingPods)
	}
	// add assumed pod
	cache.AddPod(pb_1)
	cache.UpdateSnapshot(snapshot)
	gotPendingPods = snapshot.(*PendingPodStore).GetPendingPods()
	expectedPendingPods = map[string]int{}
	if !reflect.DeepEqual(expectedPendingPods, gotPendingPods) {
		t.Errorf("expected %v but got %v", expectedPendingPods, gotPendingPods)
	}
	// update assumed -> pending
	cache.UpdatePod(pb_1, pb_2)
	cache.UpdateSnapshot(snapshot)
	gotPendingPods = snapshot.(*PendingPodStore).GetPendingPods()
	expectedPendingPods = map[string]int{
		"": 1,
	}
	if !reflect.DeepEqual(expectedPendingPods, gotPendingPods) {
		t.Errorf("expected %v but got %v", expectedPendingPods, gotPendingPods)
	}
	// remove bound pod
	cache.RemovePod(pa_2)
	cache.UpdateSnapshot(snapshot)
	gotPendingPods = snapshot.(*PendingPodStore).GetPendingPods()
	expectedPendingPods = map[string]int{
		"": 1,
	}
	if !reflect.DeepEqual(expectedPendingPods, gotPendingPods) {
		t.Errorf("expected %v but got %v", expectedPendingPods, gotPendingPods)
	}
	// remove pending pod
	cache.RemovePod(pb_2)
	cache.UpdateSnapshot(snapshot)
	gotPendingPods = snapshot.(*PendingPodStore).GetPendingPods()
	expectedPendingPods = map[string]int{}
	if !reflect.DeepEqual(expectedPendingPods, gotPendingPods) {
		t.Errorf("expected %v but got %v", expectedPendingPods, gotPendingPods)
	}
}
