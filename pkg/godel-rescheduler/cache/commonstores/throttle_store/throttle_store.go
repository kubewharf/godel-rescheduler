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
	"sync"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/api"
	reschedulercommonstores "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores"

	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/util/generationstore"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	"k8s.io/apimachinery/pkg/util/wait"
)

const Name reschedulercommonstores.StoreName = "ThrottleStore"

func (s *ThrottleStore) Name() reschedulercommonstores.StoreName {
	return Name
}

// Implement the `init` function and register the storage to the registry when the program starts.
func init() {
	reschedulercommonstores.ReschedulerGlobalRegistry.Register(
		Name,
		func(h commoncache.CacheHandler) bool {
			return true
		},
		NewCache,
		NewSnapshot)
}

var labelSelectorKeys []string = []string{"psm", podutil.PodGroupNameAnnotationKey, "name"}

type InstanceItem struct {
	time               time.Time
	movementGeneration uint
}

func (item *InstanceItem) GetTime() time.Time {
	return item.time
}

func (item *InstanceItem) GetMovementGeneration() uint {
	return item.movementGeneration
}

func (item *InstanceItem) SetTime(time time.Time) {
	item.time = time
}

type reschedulingItem struct {
	instanceItems []*InstanceItem
	generation    int64
}

func newReschedulingItem() *reschedulingItem {
	return &reschedulingItem{}
}

func (rt *reschedulingItem) GetGeneration() int64 {
	return rt.generation
}

func (rt *reschedulingItem) SetGeneration(g int64) {
	rt.generation = g
}

func (rt *reschedulingItem) Replace(newItem *reschedulingItem) {
	rt.generation = newItem.generation
	rt.instanceItems = make([]*InstanceItem, len(newItem.instanceItems))
	copy(rt.instanceItems, newItem.instanceItems)
}

type ThrottleStore struct {
	reschedulercommonstores.BaseStore
	storeType reschedulercommonstores.StoreType
	handler   commoncache.CacheHandler

	// selector label key, selector label value, rescheduling times
	stores map[string]generationstore.Store
	// selector label key, selector label value, group times
	groupStores map[string]generationstore.Store
}

func NewCache(handler commoncache.CacheHandler) reschedulercommonstores.CommonStore {
	stores := map[string]generationstore.Store{}
	groupStores := map[string]generationstore.Store{}
	for _, key := range labelSelectorKeys {
		stores[key] = generationstore.NewListStore()
		groupStores[key] = generationstore.NewListStore()
	}
	return &ThrottleStore{
		BaseStore: reschedulercommonstores.NewBaseStore(),
		storeType: reschedulercommonstores.Cache,
		handler:   handler,

		stores:      stores,
		groupStores: groupStores,
	}
}

func NewSnapshot(handler commoncache.CacheHandler) reschedulercommonstores.CommonStore {
	stores := map[string]generationstore.Store{}
	groupStores := map[string]generationstore.Store{}
	for _, key := range labelSelectorKeys {
		stores[key] = generationstore.NewRawStore()
		groupStores[key] = generationstore.NewRawStore()
	}
	return &ThrottleStore{
		BaseStore: reschedulercommonstores.NewBaseStore(),
		storeType: reschedulercommonstores.Snapshot,
		handler:   handler,

		stores:      stores,
		groupStores: groupStores,
	}
}

func (s *ThrottleStore) UpdateSnapshot(store reschedulercommonstores.CommonStore) error {
	for key, victimSetMap := range s.stores {
		updateVictimSetMap(victimSetMap, store.(*ThrottleStore).stores[key])
	}
	for key, victimSetMap := range s.groupStores {
		updateVictimSetMap(victimSetMap, store.(*ThrottleStore).groupStores[key])
	}
	return nil
}

func updateVictimSetMap(cacheStore, snapshotStore generationstore.Store) {
	cache, snapshot := framework.TransferGenerationStore(cacheStore, snapshotStore)
	cache.UpdateRawStore(
		snapshot,
		func(key string, obj generationstore.StoredObj) {
			item := obj.(*reschedulingItem)
			var existing *reschedulingItem
			if obj := snapshot.Get(key); obj != nil {
				existing = obj.(*reschedulingItem)
			} else {
				existing = newReschedulingItem()
			}
			existing.Replace(item)
			snapshot.Set(key, existing)
		},
		generationstore.DefaultCleanFunc(cache, snapshot),
	)
}

func (s *ThrottleStore) PeriodWorker(mu *sync.RWMutex) {
	go wait.Until(func() {
		s.cleanupReschedulingItems(mu, time.Now())
	}, time.Minute, s.handler.StopCh())
}

func (s *ThrottleStore) cleanupReschedulingItems(mu *sync.RWMutex, now time.Time) {
	mu.Lock()
	defer mu.Unlock()

	for _, store := range s.stores {
		cleanupEachStore(store, now)
	}
	for _, store := range s.groupStores {
		cleanupEachStore(store, now)
	}
}

func cleanupEachStore(store generationstore.Store, now time.Time) {
	store.Range(func(s string, so generationstore.StoredObj) {
		reschedulingItem := so.(*reschedulingItem)
		index := -1
		for i, eachInstance := range reschedulingItem.instanceItems {
			if now.Before(eachInstance.time.Add(time.Hour * 24)) {
				break
			}
			index = i
		}
		reschedulingItem.instanceItems = reschedulingItem.instanceItems[index+1:]
		store.Set(s, reschedulingItem)
	})
}

func (s *ThrottleStore) AssumePod(podInfo *api.CachePodInfo) error {
	s.assumeInPodStore(podInfo)
	s.assumeInGroupStore(podInfo)
	return nil
}

func (s *ThrottleStore) assumeInPodStore(podInfo *api.CachePodInfo) {
	for key, store := range s.stores {
		val := podInfo.PodInfo.Pod.Labels[key]
		if val == "" {
			continue
		}
		var existing *reschedulingItem
		obj := store.Get(val)
		if obj != nil {
			existing = obj.(*reschedulingItem)
		} else {
			existing = newReschedulingItem()
		}
		existing.instanceItems = append(existing.instanceItems, &InstanceItem{time.Now(), podInfo.MovementGeneration})
		store.Set(val, existing)
	}
}

func (s *ThrottleStore) assumeInGroupStore(podInfo *api.CachePodInfo) {
	for key, store := range s.groupStores {
		val := podInfo.PodInfo.Pod.Labels[key]
		if val == "" {
			continue
		}
		var existing *reschedulingItem
		obj := store.Get(val)
		if obj != nil {
			existing = obj.(*reschedulingItem)
		} else {
			existing = newReschedulingItem()
		}
		length := len(existing.instanceItems)
		if length == 0 {
			existing.instanceItems = append(existing.instanceItems, &InstanceItem{time.Now(), podInfo.MovementGeneration})
		} else {
			latestGeneration := existing.instanceItems[length-1].GetMovementGeneration()
			if latestGeneration == podInfo.MovementGeneration {
				existing.instanceItems[length-1] = &InstanceItem{time.Now(), podInfo.MovementGeneration}
			} else {
				existing.instanceItems = append(existing.instanceItems, &InstanceItem{time.Now(), podInfo.MovementGeneration})
			}
		}
		store.Set(val, existing)
	}
}

func (s *ThrottleStore) GetReschedulingItemsForPodThrottle(throttleItems []config.ThrottleItem) map[string]map[string][]*InstanceItem {
	return getReschedulingItems(throttleItems, s.stores)
}

func (s *ThrottleStore) GetReschedulingItemsForGroupThrottle(throttleItems []config.ThrottleItem) map[string]map[string][]*InstanceItem {
	return getReschedulingItems(throttleItems, s.groupStores)
}

func getReschedulingItems(throttleItems []config.ThrottleItem, stores map[string]generationstore.Store) map[string]map[string][]*InstanceItem {
	aggravatedThrottleItems := map[string]config.ThrottleItem{}
	for _, throttleItem := range throttleItems {
		if existingItem, ok := aggravatedThrottleItems[throttleItem.LabelSelectorKey]; ok {
			if throttleItem.ThrottleValue > existingItem.ThrottleValue {
				aggravatedThrottleItems[throttleItem.LabelSelectorKey] = throttleItem
			}
		} else {
			aggravatedThrottleItems[throttleItem.LabelSelectorKey] = throttleItem
		}
	}

	items := map[string]map[string][]*InstanceItem{}
	for _, throttleItem := range aggravatedThrottleItems {
		reschedulingItems := stores[throttleItem.LabelSelectorKey]
		if reschedulingItems == nil {
			continue
		}
		items[throttleItem.LabelSelectorKey] = map[string][]*InstanceItem{}
		reschedulingItems.Range(func(s string, so generationstore.StoredObj) {
			reschedulingItem := so.(*reschedulingItem)
			count := len(reschedulingItem.instanceItems)
			if count <= int(throttleItem.ThrottleValue) {
				items[throttleItem.LabelSelectorKey][s] = make([]*InstanceItem, count)
				copy(items[throttleItem.LabelSelectorKey][s], reschedulingItem.instanceItems)
			} else {
				items[throttleItem.LabelSelectorKey][s] = make([]*InstanceItem, throttleItem.ThrottleValue)
				copy(items[throttleItem.LabelSelectorKey][s], reschedulingItem.instanceItems[count-int(throttleItem.ThrottleValue):])
			}
		})
	}
	return items
}
