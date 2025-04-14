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
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/util"

	v1 "k8s.io/api/core/v1"
)

type throttleStore struct {
	cache         *throttleCache
	throttleItems map[string][]config.ThrottleItem
}

var _ policies.PolicyStore = &throttleStore{}

func newThrottleStore(snapshot *cache.ReschedulingSnapshot, throttleItems []config.ThrottleItem) policies.PolicyStore {
	throttleItemsMap := map[string][]config.ThrottleItem{}
	for _, item := range throttleItems {
		throttleItemsMap[item.LabelSelectorKey] = append(throttleItemsMap[item.LabelSelectorKey], item)
	}
	return &throttleStore{
		cache:         generateCache(snapshot, throttleItems),
		throttleItems: throttleItemsMap,
	}
}

func (tStore *throttleStore) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod) policies.CheckResult {
	now := time.Now()
	for key, reschedulingItems := range tStore.cache.reschedulingItems {
		val, ok := instance.Labels[key]
		if !ok {
			continue
		}
		throttleItem := util.GetThrottleItemForInstance(instance, tStore.throttleItems[key])
		if throttleItem == nil {
			continue
		}
		index := len(reschedulingItems[val])
		for i, eachTime := range reschedulingItems[val] {
			if now.After(eachTime.Add(throttleItem.ThrottleDuration.Duration)) {
				continue
			}
			index = i
			break
		}
		count := len(reschedulingItems[val]) - index
		if throttleItem.ThrottleValue <= int64(count) {
			return policies.MovedInstanceError
		}
	}
	return policies.Pass
}

func (tStore *throttleStore) AssumePod(pod *v1.Pod) {
	now := time.Now()
	for key, reschedulingItems := range tStore.cache.reschedulingItems {
		val, ok := pod.Labels[key]
		if !ok {
			continue
		}
		reschedulingItems[val] = append(reschedulingItems[val], now)
	}
}

func (tStore *throttleStore) RemovePod(pod *v1.Pod) {}

type throttleCache struct {
	reschedulingItems map[string]map[string][]time.Time
}

func generateCache(snapshot *cache.ReschedulingSnapshot, throttleItems []config.ThrottleItem) *throttleCache {
	reschedulingItems := map[string]map[string][]time.Time{}
	for labelKey, labelValMap := range snapshot.GetReschedulingItemsForPodThrottle(throttleItems) {
		if reschedulingItems[labelKey] == nil {
			reschedulingItems[labelKey] = map[string][]time.Time{}
		}
		for labelVal, reschedulerItem := range labelValMap {
			if reschedulingItems[labelKey][labelVal] == nil {
				reschedulingItems[labelKey][labelVal] = []time.Time{}
			}
			for _, item := range reschedulerItem {
				reschedulingItems[labelKey][labelVal] = append(reschedulingItems[labelKey][labelVal], item.GetTime())
			}
		}
	}
	return &throttleCache{
		reschedulingItems: reschedulingItems,
	}
}
