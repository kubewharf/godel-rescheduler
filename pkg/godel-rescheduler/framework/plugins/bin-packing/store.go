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

package binpacking

import (
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/framework/utils"
	v1 "k8s.io/api/core/v1"
)

type binPackingStore struct {
	nodes          map[string]framework.NodeInfo
	resourcesItems []config.ResourceItem
}

var _ policies.PolicyStore = &binPackingStore{}

func newBinPackingStore(snapshot *cache.ReschedulingSnapshot, resourcesItems []config.ResourceItem) policies.PolicyStore {
	nodes := map[string]framework.NodeInfo{}
	for _, n := range snapshot.NodeInfos().List() {
		if n == nil {
			continue
		}
		nodes[n.GetNodeName()] = n
	}
	return &binPackingStore{
		nodes:          nodes,
		resourcesItems: resourcesItems,
	}
}

func (bpStore *binPackingStore) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod) policies.CheckResult {
	previousNodeInfo := bpStore.nodes[previousNode]
	targetNodeInfo := bpStore.nodes[targetNode]
	previousNodeInfoClone := previousNodeInfo.Clone()
	previousNodeInfoClone.RemovePod(instance, true)
	cpuUtilizationPrevious := computeResourceUtilization(previousNodeInfoClone, bpStore.resourcesItems)
	cpuUtilizationTarget := computeResourceUtilization(targetNodeInfo, bpStore.resourcesItems)
	if cpuUtilizationPrevious > cpuUtilizationTarget {
		return policies.MovementItemError
	}
	return policies.Pass
}

func (bpStore *binPackingStore) AssumePod(pod *v1.Pod) {
	nodeName := utils.GetNodeNameFromPod(pod)
	nInfo := bpStore.nodes[nodeName]
	if nInfo == nil {
		return
	}
	nInfo.AddPod(pod)
}

func (bpStore *binPackingStore) RemovePod(pod *v1.Pod) {
	nodeName := utils.GetNodeNameFromPod(pod)
	nInfo := bpStore.nodes[nodeName]
	if nInfo == nil {
		return
	}
	nInfo.RemovePod(pod, true)
}
