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
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	reschedulercommonstores "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores"
	pendingpodstore "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores/pending_pod_store"
	throttlestore "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/commonstores/throttle_store"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"

	schedulingv1a1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/scheduling/v1alpha1"
	crdinformers "github.com/kubewharf/godel-scheduler-api/pkg/client/informers/externalversions"
	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	commonstore "github.com/kubewharf/godel-scheduler/pkg/common/store"
	schedulerframework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/cache"
	pdbstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/pdb_store"
	podgroupstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/podgroup_store"
	preemptionstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/preemption_store"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
)

const (
	DetectorStore  string = "DetectorStore"
	AlgorithmStore string = "AlgorithmStore"
)

type ReschedulingSnapshot struct {
	schedulerSnapshot        *cache.Snapshot
	policyStoresForDetector  map[string]policies.PolicyStore
	policyStoresForAlgorithm map[string]policies.PolicyStore
	storeSwitch              *CommonStoresSwitch
}

func NewEmptySnapshot(handler commoncache.CacheHandler) *ReschedulingSnapshot {
	return &ReschedulingSnapshot{
		schedulerSnapshot:        cache.NewEmptySnapshot(handler),
		policyStoresForDetector:  map[string]policies.PolicyStore{},
		policyStoresForAlgorithm: map[string]policies.PolicyStore{},

		storeSwitch: makeStoreSwitch(handler, reschedulercommonstores.Snapshot),
	}
}

func (s *ReschedulingSnapshot) AssumePod(pod *v1.Pod, gpuBinPacking bool, storeType string) error {
	state := schedulerframework.NewCycleState()
	rType, _ := podutil.GetPodResourceType(pod)
	schedulerframework.SetPodResourceTypeState(rType, state)
	s.assumePodInPolicyStore(pod, storeType)
	cachePodInfo := schedulerframework.MakeCachePodInfoWrapper().CycleState(state).
		Pod(pod.DeepCopy()).Obj()
	return s.schedulerSnapshot.AssumePod(cachePodInfo)
}

func (s *ReschedulingSnapshot) assumePodInPolicyStore(pod *v1.Pod, storeType string) {
	var policyStores map[string]policies.PolicyStore
	switch storeType {
	case DetectorStore:
		policyStores = s.policyStoresForDetector
	case AlgorithmStore:
		policyStores = s.policyStoresForAlgorithm
	}
	for _, policySnapshot := range policyStores {
		policySnapshot.AssumePod(pod)
	}
}

func (s *ReschedulingSnapshot) RemovePod(pod *v1.Pod, storeType string) error {
	s.removePodInPolicyStore(pod, storeType)
	cachePodInfo := schedulerframework.MakeCachePodInfoWrapper().
		Pod(pod.DeepCopy()).Obj()
	return s.schedulerSnapshot.ForgetPod(cachePodInfo)
}

func (s *ReschedulingSnapshot) removePodInPolicyStore(pod *v1.Pod, storeType string) {
	var policyStores map[string]policies.PolicyStore
	switch storeType {
	case DetectorStore:
		policyStores = s.policyStoresForDetector
	case AlgorithmStore:
		policyStores = s.policyStoresForAlgorithm
	}
	for _, policySnapshot := range policyStores {
		policySnapshot.RemovePod(pod)
	}
}

func (s *ReschedulingSnapshot) GetPDBItemList() []schedulerframework.PDBItem {
	store := s.schedulerSnapshot.FindStore(pdbstore.Name).(*pdbstore.PdbStore)
	return store.GetPDBItemList()
}

func (s *ReschedulingSnapshot) List() []schedulerframework.NodeInfo {
	return s.schedulerSnapshot.List()
}

func (s *ReschedulingSnapshot) Get(nodeName string) (schedulerframework.NodeInfo, error) {
	return s.schedulerSnapshot.Get(nodeName)
}

func (s *ReschedulingSnapshot) NodeInfos() schedulerframework.NodeInfoLister {
	return s.schedulerSnapshot.NodeInfos()
}

func (s *ReschedulingSnapshot) GetPodGroupInfo(podGroupName string) (*schedulingv1a1.PodGroup, error) {
	store := s.schedulerSnapshot.FindStore(podgroupstore.Name).(*podgroupstore.PodGroupStore)
	return store.GetPodGroupInfo(podGroupName)
}

func (s *ReschedulingSnapshot) SnapshotSharedLister() schedulerframework.SharedLister {
	return s.schedulerSnapshot
}

func (s *ReschedulingSnapshot) GetPDBItemListForOwner(ownerType, ownerKey string) (bool, bool, []string) {
	store := s.schedulerSnapshot.FindStore(pdbstore.Name).(*pdbstore.PdbStore)
	return store.GetPDBsForOwner(ownerType, ownerKey)
}

func (s *ReschedulingSnapshot) GetOwnerLabels(ownerType, ownerKey string) map[string]string {
	store := s.schedulerSnapshot.FindStore(pdbstore.Name).(*pdbstore.PdbStore)
	return store.GetOwnerLabels(ownerType, ownerKey)
}

func (s *ReschedulingSnapshot) GetSchedulerSnapshot() *cache.Snapshot {
	return s.schedulerSnapshot
}

func (s *ReschedulingSnapshot) NodeInfosInScope(nodeSelector map[string]string) []schedulerframework.NodeInfo {
	result := []schedulerframework.NodeInfo{}
	nodeInfos := s.schedulerSnapshot.List()
	selector := labels.SelectorFromSet(nodeSelector)
	for _, nodeInfo := range nodeInfos {
		node := nodeInfo.GetNode()
		if node == nil {
			continue
		}
		nodeLabels := node.Labels
		if !selector.Matches(labels.Set(nodeLabels)) {
			continue
		}
		result = append(result, nodeInfo)
	}
	return result
}

func (s *ReschedulingSnapshot) GetNodeInfo(nodeName string) (schedulerframework.NodeInfo, error) {
	return s.schedulerSnapshot.Get(nodeName)
}

func (s *ReschedulingSnapshot) SyncPolicyStore(policyName string, policyStore policies.PolicyStore, storeType string) {
	switch storeType {
	case DetectorStore:
		s.policyStoresForDetector[policyName] = policyStore
	case AlgorithmStore:
		s.policyStoresForAlgorithm[policyName] = policyStore
	}
}

func (s *ReschedulingSnapshot) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod, storeType string) (string, policies.CheckResult) {
	var policyStores map[string]policies.PolicyStore
	switch storeType {
	case DetectorStore:
		policyStores = s.policyStoresForDetector
	case AlgorithmStore:
		policyStores = s.policyStoresForAlgorithm
	}
	for policyName, policySnapshot := range policyStores {
		if policySnapshot == nil {
			return policyName, policies.MovementItemError
		}
		res := policySnapshot.CheckMovementItem(previousNode, targetNode, instance)
		if res == policies.Pass {
			continue
		}
		return policyName, res
	}
	return "", policies.Pass
}

func (s *ReschedulingSnapshot) GetReschedulingItemsForPodThrottle(throttleItems []config.ThrottleItem) map[string]map[string][]*throttlestore.InstanceItem {
	return s.storeSwitch.Find(throttlestore.Name).(*throttlestore.ThrottleStore).GetReschedulingItemsForPodThrottle(throttleItems)
}

func (s *ReschedulingSnapshot) GetReschedulingItemsForGroupThrottle(throttleItems []config.ThrottleItem) map[string]map[string][]*throttlestore.InstanceItem {
	return s.storeSwitch.Find(throttlestore.Name).(*throttlestore.ThrottleStore).GetReschedulingItemsForGroupThrottle(throttleItems)
}

func (s *ReschedulingSnapshot) GetPendingPods() map[string]int {
	return s.storeSwitch.Find(pendingpodstore.Name).(*pendingpodstore.PendingPodStore).GetPendingPods()
}

func (gs *ReschedulingSnapshot) FindStore(storeName commonstore.StoreName) commonstore.Store {
	return gs.schedulerSnapshot.FindStore(storeName)
}

type ReschedulingSnapshotLister struct {
	lister []*ReschedulingSnapshot
	index  int
}

func NewReschedulingSnapshotLister(size int, informerFactory informers.SharedInformerFactory, crdInformerFactory crdinformers.SharedInformerFactory) *ReschedulingSnapshotLister {
	snapshotLister := &ReschedulingSnapshotLister{
		lister: make([]*ReschedulingSnapshot, size),
	}
	for i := 0; i < size; i++ {
		handlerWrapper := commoncache.MakeCacheHandlerWrapper().PodInformer(informerFactory.Core().V1().Pods()).PodLister(informerFactory.Core().V1().Pods().Lister()).
			EnableStore(string(pdbstore.Name), string(preemptionstore.Name))
		snapshotLister.lister[i] = NewEmptySnapshot(handlerWrapper.Obj())
	}
	return snapshotLister
}

func (snapshotLister *ReschedulingSnapshotLister) Pop() *ReschedulingSnapshot {
	result := snapshotLister.lister[snapshotLister.index]
	snapshotLister.index++
	return result
}
