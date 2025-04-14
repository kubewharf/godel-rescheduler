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

package cache

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	reschedulercommonstores "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores"
	pendingpodstore "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores/pending_pod_store"
	throttlestore "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores/throttle_store"

	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
)

// ATTENTION: The stores should be called in a certain order.
var orderedReschedulerStoreNames = []reschedulercommonstores.StoreName{
	throttlestore.Name,
	pendingpodstore.Name,
}

type CommonStoresSwitch struct {
	Stores []reschedulercommonstores.CommonStore
}

type RangeFunc func(reschedulercommonstores.CommonStore) error

func (s *CommonStoresSwitch) Find(name reschedulercommonstores.StoreName) reschedulercommonstores.CommonStore {
	// When the slice is small, direct traversal will be faster than indexing in a map.
	// This is because the latter requires hash operations.
	for _, store := range s.Stores {
		if store.Name() == name {
			return store
		}
	}
	return nil
}

func (s *CommonStoresSwitch) Range(f RangeFunc) error {
	var errs []error
	for i := range s.Stores {
		if err := f(s.Stores[i]); err != nil {
			klog.ErrorS(err, "Error occurred in CommonStoresSwitch Range", "failureStore", s.Stores[i].Name(), "index", i, "total", len(s.Stores))
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (s *CommonStoresSwitch) String() string {
	var ret string
	for i := range s.Stores {
		ret += string(s.Stores[i].Name()) + ","
	}
	return ret
}

func makeStoreSwitch(handler commoncache.CacheHandler, storeType reschedulercommonstores.StoreType) *CommonStoresSwitch {
	var registry reschedulercommonstores.Registry
	if storeType == reschedulercommonstores.Cache {
		registry = reschedulercommonstores.ReschedulerGlobalRegistry.CacheRegistry
	} else {
		registry = reschedulercommonstores.ReschedulerGlobalRegistry.SnapshotRegistry
	}

	stores := make([]reschedulercommonstores.CommonStore, 0)
	for i, name := range orderedReschedulerStoreNames {
		checker, ok := reschedulercommonstores.ReschedulerGlobalRegistry.FeatureGateCheckers[name]
		if !ok || checker == nil {
			panic("Invalid commonstores registry checker")
		}
		if checker(handler) {
			newFunc, ok := registry[name]
			if !ok || newFunc == nil {
				panic("Invalid commonstores registry new function")
			}
			klog.V(4).InfoS("Registered CommonStore successfully", "storeType", storeType, "subCluster", handler.SubCluster(), "idx", i, "name", name)
			stores = append(stores, newFunc(handler))
		} else {
			klog.V(4).InfoS("Skipped register CommonStore because couldn't pass the checker", "storeType", storeType, "subCluster", handler.SubCluster(), "idx", i, "name", name)
		}
	}
	return &CommonStoresSwitch{Stores: stores}
}
