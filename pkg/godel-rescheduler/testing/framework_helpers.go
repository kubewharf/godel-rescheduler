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

package testing

import (
	"context"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	reschedulerpodutil "github.com/kubewharf/godel-rescheduler/pkg/util/pod"

	schedulingv1a1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/scheduling/v1alpha1"
	crdinformers "github.com/kubewharf/godel-scheduler-api/pkg/client/informers/externalversions"
	commonstore "github.com/kubewharf/godel-scheduler/pkg/common/store"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
)

type MockReschedulerFrameworkHandle struct {
	findNodesThatFitPod func(ctx context.Context, pod *v1.Pod, nodes []framework.NodeInfo) ([]framework.NodeInfo, framework.NodeToStatusMap, error)
	snapshot            *cache.ReschedulingSnapshot
	cache               *cache.ReschedulerCache
	scope               map[string]string
}

func NewMockReschedulerFrameworkHandle(
	findNodesThatFitPod func(ctx context.Context, pod *v1.Pod, nodes []framework.NodeInfo) ([]framework.NodeInfo, framework.NodeToStatusMap, error),
	cache *cache.ReschedulerCache,
	snapshot *cache.ReschedulingSnapshot,
	scope map[string]string,
) api.ReschedulerFrameworkHandle {
	return &MockReschedulerFrameworkHandle{
		findNodesThatFitPod: findNodesThatFitPod,
		cache:               cache,
		snapshot:            snapshot,
		scope:               scope,
	}
}

func (mrfh *MockReschedulerFrameworkHandle) FindNodesThatFitPod(ctx context.Context, pod *v1.Pod, nodes []framework.NodeInfo) ([]framework.NodeInfo, framework.NodeToStatusMap, error) {
	return mrfh.findNodesThatFitPod(ctx, pod, nodes)
}

func (mrfh *MockReschedulerFrameworkHandle) AssumePod(pod *v1.Pod, node string, storeType string) error {
	podClone := reschedulerpodutil.PrepareReschedulePod(pod)
	podClone.Annotations[podutil.AssumedNodeAnnotationKey] = node
	podClone.Spec.NodeName = node
	podClone.Annotations[podutil.MicroTopologyKey] = pod.Annotations[podutil.MicroTopologyKey]
	return mrfh.snapshot.AssumePod(podClone, false, storeType)
}

func (mrfh *MockReschedulerFrameworkHandle) RemovePod(pod *v1.Pod, storeType string) error {
	return mrfh.snapshot.RemovePod(pod, storeType)
}

func (mrfh *MockReschedulerFrameworkHandle) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod, storeType string) (string, policies.CheckResult) {
	return mrfh.snapshot.CheckMovementItem(previousNode, targetNode, instance, storeType)
}

func (mrfh *MockReschedulerFrameworkHandle) SetSnapshot(reschedulingSnapshot *cache.ReschedulingSnapshot) {
	mrfh.snapshot = reschedulingSnapshot
}

func (mrfh *MockReschedulerFrameworkHandle) GetNodeInfo(nodeName string) (framework.NodeInfo, error) {
	return mrfh.snapshot.Get(nodeName)
}

func (mrfh *MockReschedulerFrameworkHandle) NodeInfosInScope() []framework.NodeInfo {
	return mrfh.snapshot.NodeInfosInScope(mrfh.scope)
}

func (mrfh *MockReschedulerFrameworkHandle) SwitchType() framework.SwitchType {
	return 0
}

func (mrfh *MockReschedulerFrameworkHandle) SubCluster() string {
	return ""
}

func (mrfh *MockReschedulerFrameworkHandle) SchedulerName() string {
	return ""
}

func (mrfh *MockReschedulerFrameworkHandle) SnapshotSharedLister() framework.SharedLister {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) ClientSet() clientset.Interface {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) SharedInformerFactory() informers.SharedInformerFactory {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) CRDSharedInformerFactory() crdinformers.SharedInformerFactory {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) GetFrameworkForPod(*v1.Pod) (framework.SchedulerFramework, error) {
	return nil, nil
}

func (mrfh *MockReschedulerFrameworkHandle) GetPodGroupInfo(podGroupName string) (*schedulingv1a1.PodGroup, error) {
	return nil, nil
}

func (mrfh *MockReschedulerFrameworkHandle) SetPotentialVictims(node string, potentialVictims []string) {
	return
}

func (mrfh *MockReschedulerFrameworkHandle) GetPotentialVictims(node string) []string {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) GetPDBItemList() []framework.PDBItem {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) GetPDBItemListForOwner(ownerType, ownerKey string) (bool, bool, []string) {
	return false, false, nil
}

func (mrfh *MockReschedulerFrameworkHandle) GetOwnerLabels(ownerType, ownerKey string) map[string]string {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) GetPreemptionFrameworkForPod(*v1.Pod) framework.SchedulerPreemptionFramework {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) HasSearchingPlugin(string) bool {
	return false
}

func (mrfh *MockReschedulerFrameworkHandle) GetPreemptionPolicy(deployName string) string {
	return ""
}

func (mrfh *MockReschedulerFrameworkHandle) CachePreemptionPolicy(deployName string, policyName string) {
	return
}

func (mrfh *MockReschedulerFrameworkHandle) CleanupPreemptionPolicyForPodOwner() {
	return
}

func (mrfh *MockReschedulerFrameworkHandle) GetLoadAwareNodeMetricInfo(nodeName string, resourceType podutil.PodResourceType) *framework.LoadAwareNodeMetricInfo {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) GetLoadAwareNodeUsage(nodeName string, resourceType podutil.PodResourceType) *framework.LoadAwareNodeUsage {
	return nil
}

func (mrfh *MockReschedulerFrameworkHandle) FindStore(storeName commonstore.StoreName) commonstore.Store {
	return mrfh.snapshot.FindStore(storeName)
}

func (mrfh *MockReschedulerFrameworkHandle) GetPod(key string) (*v1.Pod, error) {
	return mrfh.cache.GetPod(key)
}

func (mrfh *MockReschedulerFrameworkHandle) GeneratePluginRegistryMap() {}
