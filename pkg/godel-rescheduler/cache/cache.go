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
	"sync"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/api"
	reschedulercommonstores "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores"

	nodev1alpha1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/node/v1alpha1"
	schedulingv1a1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/scheduling/v1alpha1"
	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/cache"
	katalystv1alpha1 "github.com/kubewharf/katalyst-api/pkg/apis/node/v1alpha1"
	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ReschedulerCache struct {
	mu             sync.RWMutex
	schedulerCache cache.SchedulerCache

	storeSwitch *CommonStoresSwitch
}

func New(handler commoncache.CacheHandler) *ReschedulerCache {
	return &ReschedulerCache{
		schedulerCache: cache.New(handler),

		storeSwitch: makeStoreSwitch(handler, reschedulercommonstores.Cache),
	}
}

func (cache *ReschedulerCache) AssumePod(podInfo *api.CachePodInfo) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.storeSwitch.Range(func(cs reschedulercommonstores.CommonStore) error {
		return cs.AssumePod(podInfo)
	})
}

/*
func (cache *ReschedulerCache) ForgetPod(podInfo *api.CachePodInfo) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.ForgetPod(podInfo)
}
*/

func (cache *ReschedulerCache) AddPod(pod *v1.Pod) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.storeSwitch.Range(func(cs reschedulercommonstores.CommonStore) error {
		return cs.AddPod(pod)
	})
	return cache.schedulerCache.AddPod(pod)
}

func (cache *ReschedulerCache) UpdatePod(oldPod, newPod *v1.Pod) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.storeSwitch.Range(func(cs reschedulercommonstores.CommonStore) error {
		return cs.UpdatePod(oldPod, newPod)
	})
	return cache.schedulerCache.UpdatePod(oldPod, newPod)
}

func (cache *ReschedulerCache) RemovePod(pod *v1.Pod) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.storeSwitch.Range(func(cs reschedulercommonstores.CommonStore) error {
		return cs.RemovePod(pod)
	})
	return cache.schedulerCache.DeletePod(pod)
}

func (cache *ReschedulerCache) AddNode(node *v1.Node) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.AddNode(node)
}

func (cache *ReschedulerCache) AddNMNode(nmNode *nodev1alpha1.NMNode) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.AddNMNode(nmNode)
}

func (cache *ReschedulerCache) AddCNR(cnr *katalystv1alpha1.CustomNodeResource) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.AddCNR(cnr)
}

func (cache *ReschedulerCache) UpdateNode(oldNode, newNode *v1.Node) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.UpdateNode(oldNode, newNode)
}

func (cache *ReschedulerCache) UpdateNMNode(oldNMNode, newNMNode *nodev1alpha1.NMNode) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.UpdateNMNode(oldNMNode, newNMNode)
}

func (cache *ReschedulerCache) UpdateCNR(oldCNR, newCNR *katalystv1alpha1.CustomNodeResource) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.UpdateCNR(oldCNR, newCNR)
}

// RemoveNode removes a node from the cache's tree.
// The node might still have pods because their deletion events didn't arrive
// yet. Those pods are considered removed from the cache, being the node tree
// the source of truth.
// However, we keep a ghost node with the list of pods until all pod deletion
// events have arrived. A ghost node is skipped from snapshots.
func (cache *ReschedulerCache) RemoveNode(node *v1.Node) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.DeleteNode(node)
}

func (cache *ReschedulerCache) RemoveNMNode(nmNode *nodev1alpha1.NMNode) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.DeleteNMNode(nmNode)
}

// RemoveCNR removes custom node resource.
// The node might be still in the node tree because their deletion events didn't arrive yet.
func (cache *ReschedulerCache) RemoveCNR(cnr *katalystv1alpha1.CustomNodeResource) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.DeleteCNR(cnr)
}

func (cache *ReschedulerCache) AddPodGroup(podGroup *schedulingv1a1.PodGroup) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.AddPodGroup(podGroup)
}

func (cache *ReschedulerCache) RemovePodGroup(podGroup *schedulingv1a1.PodGroup) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.DeletePodGroup(podGroup)
}

func (cache *ReschedulerCache) UpdatePodGroup(oldPodGroup, newPodGroup *schedulingv1a1.PodGroup) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.UpdatePodGroup(oldPodGroup, newPodGroup)
}

func (cache *ReschedulerCache) AddPDB(pdb *policy.PodDisruptionBudget) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.AddPDB(pdb)
}

func (cache *ReschedulerCache) UpdatePDB(oldPdb, newPdb *policy.PodDisruptionBudget) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.UpdatePDB(oldPdb, newPdb)
}

func (cache *ReschedulerCache) DeletePDB(pdb *policy.PodDisruptionBudget) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.DeletePDB(pdb)
}

func (cache *ReschedulerCache) AddOwner(ownerType, key string, labels map[string]string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.AddOwner(ownerType, key, labels)
}

func (cache *ReschedulerCache) UpdateOwner(ownerType, key string, oldLabels, newLabels map[string]string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.UpdateOwner(ownerType, key, oldLabels, newLabels)
}

func (cache *ReschedulerCache) DeleteOwner(ownerType, key string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.DeleteOwner(ownerType, key)
}

func (cache *ReschedulerCache) AddMovement(movement *schedulingv1a1.Movement) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.AddMovement(movement)
}

func (cache *ReschedulerCache) UpdateMovement(oldMovement, newMovement *schedulingv1a1.Movement) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.UpdateMovement(oldMovement, newMovement)
}

func (cache *ReschedulerCache) RemoveMovement(movement *schedulingv1a1.Movement) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.schedulerCache.DeleteMovement(movement)
}

func (cache *ReschedulerCache) UpdateSnapshots(snapshotLister *ReschedulingSnapshotLister) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	snapshotLister.index = 0
	for _, snapshot := range snapshotLister.lister {
		if err := cache.updateSnapshot(snapshot); err != nil {
			return err
		}
	}
	return nil
}

func (cache *ReschedulerCache) UpdateSnapshot(snapshot *ReschedulingSnapshot) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.updateSnapshot(snapshot)
}

func (cache *ReschedulerCache) updateSnapshot(snapshot *ReschedulingSnapshot) error {
	if err := cache.storeSwitch.Range(
		func(cs reschedulercommonstores.CommonStore) error {
			if s := snapshot.storeSwitch.Find(cs.Name()); s != nil {
				return cs.UpdateSnapshot(s)
			}
			return nil
		}); err != nil {
		return err
	}

	return cache.schedulerCache.UpdateSnapshot(snapshot.GetSchedulerSnapshot())
}

func (cache *ReschedulerCache) GetPod(key string) (*v1.Pod, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID(key),
		},
	}
	return cache.schedulerCache.GetPod(pod)
}
