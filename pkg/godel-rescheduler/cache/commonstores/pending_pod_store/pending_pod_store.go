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
	reschedulercommonstores "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores"
	"github.com/kubewharf/godel-rescheduler/pkg/util"

	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const Name reschedulercommonstores.StoreName = "PendingPod"

func (s *PendingPodStore) Name() reschedulercommonstores.StoreName {
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

type PendingPodStore struct {
	reschedulercommonstores.BaseStore
	storeType reschedulercommonstores.StoreType
	handler   commoncache.CacheHandler

	// key is nodeLevel, value is pending pods name set
	pendingPods map[string]sets.String
}

func NewCache(handler commoncache.CacheHandler) reschedulercommonstores.CommonStore {
	return &PendingPodStore{
		BaseStore: reschedulercommonstores.NewBaseStore(),
		storeType: reschedulercommonstores.Cache,
		handler:   handler,

		pendingPods: map[string]sets.String{},
	}
}

func NewSnapshot(handler commoncache.CacheHandler) reschedulercommonstores.CommonStore {
	return &PendingPodStore{
		BaseStore: reschedulercommonstores.NewBaseStore(),
		storeType: reschedulercommonstores.Snapshot,
		handler:   handler,

		pendingPods: map[string]sets.String{},
	}
}

func (s *PendingPodStore) UpdateSnapshot(store reschedulercommonstores.CommonStore) error {
	snapshot := store.(*PendingPodStore)
	snapshot.pendingPods = make(map[string]sets.String)
	for nodeLevel, pods := range s.pendingPods {
		snapshot.pendingPods[nodeLevel] = sets.NewString(pods.UnsortedList()...)
	}
	return nil
}

func (s *PendingPodStore) AddPod(pod *corev1.Pod) error {
	if podutil.AssumedPod(pod) || podutil.BoundPod(pod) {
		return nil
	}
	subCluster := util.GetSubClusterForPod(pod)
	podKey := podutil.GeneratePodKey(pod)
	if s.pendingPods[subCluster] == nil {
		s.pendingPods[subCluster] = sets.NewString()
	}
	s.pendingPods[subCluster].Insert(podKey)
	return nil
}

func (s *PendingPodStore) UpdatePod(oldPod, newPod *corev1.Pod) error {
	s.RemovePod(oldPod)
	s.AddPod(newPod)
	return nil
}

func (s *PendingPodStore) RemovePod(pod *corev1.Pod) error {
	subCluster := util.GetSubClusterForPod(pod)
	podKey := podutil.GeneratePodKey(pod)
	s.pendingPods[subCluster].Delete(podKey)
	if s.pendingPods[subCluster].Len() == 0 {
		delete(s.pendingPods, subCluster)
	}
	return nil
}

func (s *PendingPodStore) GetPendingPods() map[string]int {
	pendingPods := map[string]int{}
	for subCluster, pods := range s.pendingPods {
		pendingPods[subCluster] = pods.Len()
	}
	return pendingPods
}
