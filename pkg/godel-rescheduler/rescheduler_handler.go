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

package godel_rescheduler

import (
	"context"
	"sync"
	"sync/atomic"

	reschedulercache "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/runtime"
	"github.com/kubewharf/godel-rescheduler/pkg/util/parallelize"
	reschedulerpodutil "github.com/kubewharf/godel-rescheduler/pkg/util/pod"

	schedulingv1a1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/scheduling/v1alpha1"
	godelclient "github.com/kubewharf/godel-scheduler-api/pkg/client/clientset/versioned"
	crdinformers "github.com/kubewharf/godel-scheduler-api/pkg/client/informers/externalversions"
	commonstore "github.com/kubewharf/godel-scheduler/pkg/common/store"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	schedulerframework "github.com/kubewharf/godel-scheduler/pkg/scheduler/framework"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type ReschedulerForPolicy struct {
	rescheduler         *Rescheduler
	defaultPodScheduler *runtime.PodScheduler
	podSchedulerMap     map[string]*runtime.PodScheduler
	snapshot            *reschedulercache.ReschedulingSnapshot
	scope               map[string]string
}

func NewReschedulerForPolicy(
	rescheduler *Rescheduler,
	defaultPodScheduler *runtime.PodScheduler,
	podSchedulerMap map[string]*runtime.PodScheduler,
	snapshot *reschedulercache.ReschedulingSnapshot,
	scope map[string]string,
) *ReschedulerForPolicy {
	reschedulerForPolicy := &ReschedulerForPolicy{
		rescheduler:         rescheduler,
		defaultPodScheduler: defaultPodScheduler,
		podSchedulerMap:     podSchedulerMap,
		snapshot:            snapshot,
		scope:               scope,
	}
	return reschedulerForPolicy
}

func (rs *ReschedulerForPolicy) GeneratePluginRegistryMap() {
	rs.podSchedulerMap = map[string]*runtime.PodScheduler{}
	for subCluster, subClusterConfig := range rs.rescheduler.subClusterConfigs {
		rs.podSchedulerMap[subCluster] = rs.generatePluginRegistry(subClusterConfig.BasePlugins, subClusterConfig.PluginConfigs)
	}
	rs.defaultPodScheduler = rs.generatePluginRegistry(rs.rescheduler.defaultSubClusterConfig.BasePlugins, rs.rescheduler.defaultSubClusterConfig.PluginConfigs)
}

func (rs *ReschedulerForPolicy) generatePluginRegistry(basePlugins framework.PluginCollectionSet, pluginConfigs []schedulerconfig.PluginConfig) *runtime.PodScheduler {
	pluginConfigMap := map[string]*schedulerconfig.PluginConfig{}
	for i, pluginConfig := range pluginConfigs {
		pluginConfigMap[pluginConfig.Name] = &pluginConfigs[i]
	}

	pluginRegistry, err := schedulerframework.NewPluginsRegistry(rs.rescheduler.schedulerRegistry, pluginConfigMap, rs)
	if err != nil {
		klog.Fatalf("failed to initialize GodelRescheduler %v", err.Error())
	}
	if pluginRegistry == nil {
		klog.Fatalf("plugins registry is not defined, failed to initialize GodelScheduler")
	}

	return runtime.NewPodScheduler(pluginRegistry, basePlugins, pluginConfigMap)
}

func (rs *ReschedulerForPolicy) FindNodesThatFitPod(ctx context.Context, pod *v1.Pod, nodes []framework.NodeInfo) ([]framework.NodeInfo, framework.NodeToStatusMap, error) {
	fwk, err := rs.getFrameworkForPod(pod)
	if err != nil {
		return nil, nil, err
	}

	// init cycle state
	state, err := fwk.InitCycleState(pod)
	if err != nil {
		// This shouldn't happen, because we only schedule pods having annotations set
		return nil, nil, err
	}

	filteredNodesStatuses := make(framework.NodeToStatusMap)
	// Run "prefilter" plugins.
	s := fwk.RunPreFilterPlugins(ctx, state, pod)
	if !s.IsSuccess() {
		if !s.IsUnschedulable() {
			return nil, nil, s.AsError()
		}
		for _, n := range nodes {
			filteredNodesStatuses[n.GetNodeName()] = s
		}
		return nil, filteredNodesStatuses, nil
	}

	feasibleNodes, err := findNodesThatPassFilters(ctx, fwk, state, pod, filteredNodesStatuses, nodes)
	if err != nil {
		return nil, nil, err
	}
	return feasibleNodes, filteredNodesStatuses, nil
}

func findNodesThatPassFilters(
	ctx context.Context,
	f framework.SchedulerFramework,
	state *framework.CycleState,
	pod *v1.Pod,
	statuses framework.NodeToStatusMap,
	nodes []framework.NodeInfo) ([]framework.NodeInfo, error) {
	size := len(nodes)
	if size == 0 {
		return nil, nil
	}

	errCh := parallelize.NewErrorChannel()
	feasibleNodes := make([]framework.NodeInfo, size)
	var statusesLock sync.Mutex
	var feasibleNodesLen int32
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	checkNode := func(i int) {
		// We check the nodes starting from where we left off in the previous scheduling cycle,
		// this is to make sure all nodes have the same chance of being examined across pods.
		nodeInfo := nodes[i]
		fits, status, err := podPassesFiltersOnNode(ctx, f, state, pod, nodeInfo)
		if err != nil {
			klog.Errorf("error happens with filters for pod on nodes %v, %v", nodeInfo.GetNodeName(), err)
			errCh.SendErrorWithCancel(err, cancel)
			return
		}
		if fits {
			length := atomic.AddInt32(&feasibleNodesLen, 1)
			feasibleNodes[length-1] = nodeInfo
		} else {
			statusesLock.Lock()
			if !status.IsSuccess() {
				statuses[nodeInfo.GetNodeName()] = status
			}
			statusesLock.Unlock()
		}
	}

	parallelize.Until(ctx, size, checkNode)
	if err := errCh.ReceiveError(); err != nil {
		klog.Errorf("failed to get feasible nodes, %v", err)
		return nil, err
	}
	return feasibleNodes[0:feasibleNodesLen], nil
}

func podPassesFiltersOnNode(
	ctx context.Context,
	fw framework.SchedulerFramework,
	state *framework.CycleState,
	pod *v1.Pod,
	info framework.NodeInfo,
) (bool, *framework.Status, error) {
	var status *framework.Status
	statusMap := fw.RunFilterPlugins(ctx, state, pod, info)
	status = statusMap.Merge()
	if !status.IsSuccess() && !status.IsUnschedulable() {
		return false, status, status.AsError()
	}
	return status.IsSuccess(), status, nil
}

func (rs *ReschedulerForPolicy) AssumePod(pod *v1.Pod, node string, storeType string) error {
	fwk, err := rs.getFrameworkForPod(pod)
	if err != nil {
		return err
	}
	shareGPUBinPacking := fwk.ListPlugins()[framework.ScorePhase].Has("ShareGPUBinPacking")
	podClone := reschedulerpodutil.PrepareReschedulePod(pod)
	podClone.Annotations[podutil.AssumedNodeAnnotationKey] = node
	podClone.Spec.NodeName = node
	podClone.Annotations[podutil.MicroTopologyKey] = pod.Annotations[podutil.MicroTopologyKey]
	return rs.snapshot.AssumePod(podClone, shareGPUBinPacking, storeType)
}

func (rs *ReschedulerForPolicy) RemovePod(pod *v1.Pod, storeType string) error {
	return rs.snapshot.RemovePod(pod, storeType)
}

func (rs *ReschedulerForPolicy) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod, storeType string) (string, policies.CheckResult) {
	return rs.snapshot.CheckMovementItem(previousNode, targetNode, instance, storeType)
}

func (rs *ReschedulerForPolicy) SetSnapshot(reschedulingSnapshot *reschedulercache.ReschedulingSnapshot) {
	rs.snapshot = reschedulingSnapshot
}

func (rs *ReschedulerForPolicy) GetNodeInfo(nodeName string) (framework.NodeInfo, error) {
	return rs.snapshot.Get(nodeName)
}

func (rs *ReschedulerForPolicy) NodeInfosInScope() []framework.NodeInfo {
	return rs.snapshot.NodeInfosInScope(rs.scope)
}

func (gs *ReschedulerForPolicy) getFrameworkForPod(pod *v1.Pod) (framework.SchedulerFramework, error) {
	var podScheduler *runtime.PodScheduler
	subClusterVal := pod.Spec.NodeSelector[schedulerconfig.DefaultSubClusterKey]
	if subClusterVal == "" {
		podScheduler = gs.defaultPodScheduler
	} else {
		if subClusterPodScheduler, ok := gs.podSchedulerMap[subClusterVal]; ok {
			podScheduler = subClusterPodScheduler
		} else {
			podScheduler = gs.defaultPodScheduler
		}
	}
	return podScheduler.GetFrameworkForPod(pod)
}

func (gs *ReschedulerForPolicy) GetPod(key string) (*v1.Pod, error) {
	return gs.rescheduler.cache.GetPod(key)
}

func (gs *ReschedulerForPolicy) FindStore(storeName commonstore.StoreName) commonstore.Store {
	return gs.snapshot.FindStore(storeName)
}

// ------------------------ scheduler function ------------------------
func (rs *ReschedulerForPolicy) SwitchType() framework.SwitchType {
	return framework.DefaultSubClusterSwitchType
}

func (rs *ReschedulerForPolicy) SubCluster() string {
	return framework.DefaultSubCluster
}

func (rs *ReschedulerForPolicy) ClientSet() clientset.Interface {
	return rs.rescheduler.client
}

func (rs *ReschedulerForPolicy) CrdClientSet() godelclient.Interface {
	return rs.rescheduler.crdClient
}

func (rs *ReschedulerForPolicy) SharedInformerFactory() informers.SharedInformerFactory {
	return rs.rescheduler.informerFactory
}

func (rs *ReschedulerForPolicy) CRDSharedInformerFactory() crdinformers.SharedInformerFactory {
	return rs.rescheduler.crdInformerFactory
}

func (rs *ReschedulerForPolicy) GetPodGroupInfo(podGroupName string) (*schedulingv1a1.PodGroup, error) {
	return rs.snapshot.GetPodGroupInfo(podGroupName)
}

func (rs *ReschedulerForPolicy) SchedulerName() string {
	return config.DefaultSchedulerName
}

func (rs *ReschedulerForPolicy) SnapshotSharedLister() framework.SharedLister {
	return rs.snapshot.SnapshotSharedLister()
}

func (rs *ReschedulerForPolicy) GetPDBItemList() []framework.PDBItem {
	return rs.snapshot.GetPDBItemList()
}

func (rs *ReschedulerForPolicy) GetPreemptMinIntervalSeconds() int64 {
	return 0
}

func (rs *ReschedulerForPolicy) GetPreemptMinReplicaNum() int64 {
	return 0
}

func (rs *ReschedulerForPolicy) GetPreemptThrottleValue() int64 {
	return 0
}

func (rs *ReschedulerForPolicy) SetPotentialVictims(_ string, _ []string) {}

func (rs *ReschedulerForPolicy) GetPotentialVictims(node string) []string {
	return nil
}

func (gs *ReschedulerForPolicy) GetFrameworkForPod(_ *v1.Pod) (framework.SchedulerFramework, error) {
	return nil, nil
}

func (rs *ReschedulerForPolicy) SetFrameworkForPod(f framework.SchedulerFramework) {}

func (rs *ReschedulerForPolicy) GetPDBItemListForOwner(ownerType, ownerKey string) (bool, bool, []string) {
	return rs.snapshot.GetPDBItemListForOwner(ownerType, ownerKey)
}

func (rs *ReschedulerForPolicy) CleanupPreemptionPolicyForPodOwner() {}

// Note: The function's underlying access is Snapshot, Snapshot operations are lock-free.
func (rs *ReschedulerForPolicy) GetOwnerLabels(ownerType, ownerKey string) map[string]string {
	return rs.snapshot.GetOwnerLabels(ownerType, ownerKey)
}

func (rs *ReschedulerForPolicy) GetPreemptionFrameworkForPod(*v1.Pod) framework.SchedulerPreemptionFramework {
	return nil
}

func (rs *ReschedulerForPolicy) HasSearchingPlugin(string) bool {
	return false
}

func (rs *ReschedulerForPolicy) GetPreemptionPolicy(deployName string) string {
	return ""
}

func (rs *ReschedulerForPolicy) CachePreemptionPolicy(deployName string, policyName string) {}

func (rs *ReschedulerForPolicy) GetLoadAwareNodeMetricInfo(nodeName string, resourceType podutil.PodResourceType) *framework.LoadAwareNodeMetricInfo {
	return nil
}

func (rs *ReschedulerForPolicy) GetLoadAwareNodeUsage(nodeName string, resourceType podutil.PodResourceType) *framework.LoadAwareNodeUsage {
	return nil
}
