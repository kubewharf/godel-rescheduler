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

package throttlestore

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/api"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	"github.com/kubewharf/godel-scheduler/pkg/util"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pointer "k8s.io/utils/ptr"
)

func TestUpdateSnapshot(t *testing.T) {
	cache := NewCache(nil)
	snapshot := NewSnapshot(nil)
	throttleItems := []config.ThrottleItem{
		{
			LabelSelectorKey: "psm",
			ThrottleDuration: metav1.Duration{Duration: time.Hour},
			ThrottleValue:    3,
		},
		{
			LabelSelectorKey: podutil.PodGroupNameAnnotationKey,
			ThrottleDuration: metav1.Duration{Duration: time.Hour},
			ThrottleValue:    3,
		},
		{
			LabelSelectorKey: "name",
			ThrottleDuration: metav1.Duration{Duration: time.Hour},
			ThrottleValue:    3,
			CanBePreempted:   pointer.To(false),
		},
	}
	// assume in cache
	p0 := testing_helper.MakePod().Namespace("p0").Name("p0").UID("p0").
		Annotation(util.CanBePreemptedAnnotationKey, util.CanBePreempted).
		Label("name", "a").
		Label("psm", "a").
		Label(podutil.PodGroupNameAnnotationKey, "pg").Obj()
	p1 := testing_helper.MakePod().Namespace("p1").Name("p1").UID("p1").
		Annotation(util.CanBePreemptedAnnotationKey, util.CannotBePreempted).
		Label("name", "a").
		Label("psm", "a").
		Label(podutil.PodGroupNameAnnotationKey, "pg").Obj()
	p2 := testing_helper.MakePod().Namespace("p2").Name("p2").UID("p2").
		Annotation(util.CanBePreemptedAnnotationKey, util.CannotBePreempted).
		Label("name", "b").
		Label("psm", "b").Obj()
	p3 := testing_helper.MakePod().Namespace("p3").Name("p3").UID("p3").
		Label(podutil.PodGroupNameAnnotationKey, "pg").Obj()
	pInfo0 := framework.MakeCachePodInfoWrapper().Pod(p0).Obj()
	pInfo1 := framework.MakeCachePodInfoWrapper().Pod(p1).Obj()
	pInfo2 := framework.MakeCachePodInfoWrapper().Pod(p2).Obj()
	pInfo3 := framework.MakeCachePodInfoWrapper().Pod(p3).Obj()
	reschedPodInfo0 := api.MakeCachePodInfoWrapper().PodInfo(pInfo0).MovementGeneration(0).Obj()
	reschedPodInfo1 := api.MakeCachePodInfoWrapper().PodInfo(pInfo1).MovementGeneration(0).Obj()
	reschedPodInfo2 := api.MakeCachePodInfoWrapper().PodInfo(pInfo2).MovementGeneration(1).Obj()
	reschedPodInfo3 := api.MakeCachePodInfoWrapper().PodInfo(pInfo3).MovementGeneration(1).Obj()
	cache.AssumePod(reschedPodInfo0)
	cache.AssumePod(reschedPodInfo1)
	cache.AssumePod(reschedPodInfo2)
	cache.AssumePod(reschedPodInfo3)

	cache.UpdateSnapshot(snapshot)
	// for pod throttle
	gotItemCounts := getItemCounts(snapshot.(*ThrottleStore).GetReschedulingItemsForPodThrottle(throttleItems), throttleItems)
	expectedItemCounts := map[string]map[string][]uint{
		"psm": {
			"a": []uint{0, 0},
			"b": []uint{1},
		},
		podutil.PodGroupNameAnnotationKey: {
			"pg": []uint{0, 0, 1},
		},
		"name": {
			"a": []uint{0, 0},
			"b": []uint{1},
		},
	}
	if !reflect.DeepEqual(expectedItemCounts, gotItemCounts) {
		t.Errorf("expected %v but got %v", expectedItemCounts, gotItemCounts)
	}
	// for group throttle
	gotItemCounts = getItemCounts(snapshot.(*ThrottleStore).GetReschedulingItemsForGroupThrottle(throttleItems), throttleItems)
	expectedItemCounts = map[string]map[string][]uint{
		"psm": {
			"a": []uint{0},
			"b": []uint{1},
		},
		podutil.PodGroupNameAnnotationKey: {
			"pg": []uint{0, 1},
		},
		"name": {
			"a": []uint{0},
			"b": []uint{1},
		},
	}
	if !reflect.DeepEqual(expectedItemCounts, gotItemCounts) {
		t.Errorf("expected %v but got %v", expectedItemCounts, gotItemCounts)
	}

	// assume in snapshot
	p4 := testing_helper.MakePod().Namespace("p4").Name("p4").UID("p4").
		Annotation(util.CanBePreemptedAnnotationKey, util.CanBePreempted).
		Label("name", "a").
		Label("psm", "a").
		Label(podutil.PodGroupNameAnnotationKey, "pg").Obj()
	pInfo4 := framework.MakeCachePodInfoWrapper().Pod(p4).Obj()
	reschedPodInfo4 := api.MakeCachePodInfoWrapper().PodInfo(pInfo4).MovementGeneration(3).Obj()
	snapshot.AssumePod(reschedPodInfo4)
	p5 := testing_helper.MakePod().Namespace("p5").Name("p5").UID("p5").
		Annotation(util.CanBePreemptedAnnotationKey, util.CannotBePreempted).
		Label("name", "a").
		Label("psm", "a").
		Label(podutil.PodGroupNameAnnotationKey, "pg").Obj()
	pInfo5 := framework.MakeCachePodInfoWrapper().Pod(p5).Obj()
	reschedPodInfo5 := api.MakeCachePodInfoWrapper().PodInfo(pInfo5).MovementGeneration(4).Obj()
	snapshot.AssumePod(reschedPodInfo5)
	// for pod throttle
	gotItemCounts = getItemCounts(snapshot.(*ThrottleStore).GetReschedulingItemsForPodThrottle(throttleItems), throttleItems)
	expectedItemCounts = map[string]map[string][]uint{
		"psm": {
			"a": {0, 3, 4},
			"b": {1},
		},
		podutil.PodGroupNameAnnotationKey: {
			"pg": {1, 3, 4},
		},
		"name": {
			"a": {0, 3, 4},
			"b": {1},
		},
	}
	if !reflect.DeepEqual(expectedItemCounts, gotItemCounts) {
		t.Errorf("expected %v but got %v", expectedItemCounts, gotItemCounts)
	}
	// for group throttle
	gotItemCounts = getItemCounts(snapshot.(*ThrottleStore).GetReschedulingItemsForGroupThrottle(throttleItems), throttleItems)
	expectedItemCounts = map[string]map[string][]uint{
		"psm": {
			"a": {0, 3, 4},
			"b": {1},
		},
		podutil.PodGroupNameAnnotationKey: {
			"pg": {1, 3, 4},
		},
		"name": {
			"a": {0, 3, 4},
			"b": {1},
		},
	}
	if !reflect.DeepEqual(expectedItemCounts, gotItemCounts) {
		t.Errorf("expected %v but got %v", expectedItemCounts, gotItemCounts)
	}

	cache.UpdateSnapshot(snapshot)
	// for pod throttle
	gotItemCounts = getItemCounts(snapshot.(*ThrottleStore).GetReschedulingItemsForPodThrottle(throttleItems), throttleItems)
	expectedItemCounts = map[string]map[string][]uint{
		"psm": {
			"a": []uint{0, 0},
			"b": []uint{1},
		},
		podutil.PodGroupNameAnnotationKey: {
			"pg": []uint{0, 0, 1},
		},
		"name": {
			"a": []uint{0, 0},
			"b": []uint{1},
		},
	}
	if !reflect.DeepEqual(expectedItemCounts, gotItemCounts) {
		t.Errorf("expected %v but got %v", expectedItemCounts, gotItemCounts)
	}

	// for group throttle
	gotItemCounts = getItemCounts(snapshot.(*ThrottleStore).GetReschedulingItemsForGroupThrottle(throttleItems), throttleItems)
	expectedItemCounts = map[string]map[string][]uint{
		"psm": {
			"a": []uint{0},
			"b": []uint{1},
		},
		podutil.PodGroupNameAnnotationKey: {
			"pg": []uint{0, 1},
		},
		"name": {
			"a": []uint{0},
			"b": []uint{1},
		},
	}
	if !reflect.DeepEqual(expectedItemCounts, gotItemCounts) {
		t.Errorf("expected %v but got %v", expectedItemCounts, gotItemCounts)
	}
}

func getItemCounts(gotItems map[string]map[string][]*InstanceItem, throttleItems []config.ThrottleItem) map[string]map[string][]uint {
	gotItemCounts := map[string]map[string][]uint{}
	for key, items := range gotItems {
		gotItemCounts[key] = map[string][]uint{}
		for val, times := range items {
			gotItemCounts[key][val] = []uint{}
			for _, time := range times {
				gotItemCounts[key][val] = append(gotItemCounts[key][val], time.movementGeneration)
			}
		}
	}
	return gotItemCounts
}

func TestCleanupReschedulingItems(t *testing.T) {
	cache := NewCache(nil).(*ThrottleStore)
	now := time.Now()

	reschedulingItems := cache.stores["psm"]
	reschedulingItems.Set("a", &reschedulingItem{
		instanceItems: []*InstanceItem{
			{now.Add(-time.Hour * 25), 0},
			{now.Add(-time.Hour*24 - time.Minute), 0},
			{now.Add(-time.Hour*24 + time.Minute), 0},
			{now.Add(-time.Hour * 23), 0},
		},
	})
	reschedulingItems = cache.groupStores["psm"]
	reschedulingItems.Set("a", &reschedulingItem{
		instanceItems: []*InstanceItem{
			{now.Add(-time.Hour * 25), 0},
			{now.Add(-time.Hour*24 - time.Minute), 1},
			{now.Add(-time.Hour*24 + time.Minute), 2},
			{now.Add(-time.Hour * 23), 3},
		},
	})

	reschedulingItems = cache.stores[podutil.PodGroupNameAnnotationKey]
	reschedulingItems.Set("pg", &reschedulingItem{
		instanceItems: []*InstanceItem{
			{now.Add(-time.Hour * 25), 0},
			{now.Add(-time.Hour*24 - time.Minute), 0},
			{now.Add(-time.Hour*24 + time.Minute), 0},
			{now.Add(-time.Hour * 23), 0},
		},
	})
	reschedulingItems = cache.groupStores[podutil.PodGroupNameAnnotationKey]
	reschedulingItems.Set("pg", &reschedulingItem{
		instanceItems: []*InstanceItem{
			{now.Add(-time.Hour * 25), 0},
			{now.Add(-time.Hour*24 - time.Minute), 1},
			{now.Add(-time.Hour*24 + time.Minute), 2},
			{now.Add(-time.Hour * 23), 3},
		},
	})

	cache.cleanupReschedulingItems(&sync.RWMutex{}, now)

	expectedReschedulingItems := map[string]map[string][]*InstanceItem{
		"psm": {
			"a": {
				{now.Add(-time.Hour*24 + time.Minute), 0},
				{now.Add(-time.Hour * 23), 0},
			},
		},
		podutil.PodGroupNameAnnotationKey: {
			"pg": {
				{now.Add(-time.Hour*24 + time.Minute), 0},
				{now.Add(-time.Hour * 23), 0},
			},
		},
	}
	gotReschedulingItems := cache.GetReschedulingItemsForPodThrottle([]config.ThrottleItem{
		{
			LabelSelectorKey: "psm",
			ThrottleDuration: metav1.Duration{Duration: time.Hour},
			ThrottleValue:    100,
		},
		{
			LabelSelectorKey: podutil.PodGroupNameAnnotationKey,
			ThrottleDuration: metav1.Duration{Duration: time.Hour},
			ThrottleValue:    100,
		},
	})
	if !reflect.DeepEqual(expectedReschedulingItems, gotReschedulingItems) {
		t.Errorf("expected %v but got %v", expectedReschedulingItems, gotReschedulingItems)
	}

	expectedReschedulingItems = map[string]map[string][]*InstanceItem{
		"psm": {
			"a": {
				{now.Add(-time.Hour*24 + time.Minute), 2},
				{now.Add(-time.Hour * 23), 3},
			},
		},
		podutil.PodGroupNameAnnotationKey: {
			"pg": {
				{now.Add(-time.Hour*24 + time.Minute), 2},
				{now.Add(-time.Hour * 23), 3},
			},
		},
	}
	gotReschedulingItems = cache.GetReschedulingItemsForGroupThrottle([]config.ThrottleItem{
		{
			LabelSelectorKey: "psm",
			ThrottleDuration: metav1.Duration{Duration: time.Hour},
			ThrottleValue:    100,
		},
		{
			LabelSelectorKey: podutil.PodGroupNameAnnotationKey,
			ThrottleDuration: metav1.Duration{Duration: time.Hour},
			ThrottleValue:    100,
		},
	})
	if !reflect.DeepEqual(expectedReschedulingItems, gotReschedulingItems) {
		t.Errorf("expected %v but got %v", expectedReschedulingItems, gotReschedulingItems)
	}
}
