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
	"context"
	"fmt"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	reschedulercache "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	framework "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	reschedulerpodutil "github.com/kubewharf/godel-rescheduler/pkg/util/pod"

	godel_framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
)

const (
	Name = "BinPacking"
)

type Operator string

type BinPacking struct {
	Args *config.BinPackingArgs
	rfh  framework.ReschedulerFrameworkHandle
}

var (
	_ api.DetectorPlugin          = &BinPacking{}
	_ api.AlgorithmPlugin         = &BinPacking{}
	_ api.PolicyStoreSyncerPlugin = &BinPacking{}
)

func New(plArgs apiruntime.Object, rfh framework.ReschedulerFrameworkHandle) (godel_framework.Plugin, error) {
	args, ok := plArgs.(*config.BinPackingArgs)
	if !ok {
		return nil, fmt.Errorf("Failed to parse plugin args for %s: %v", Name, plArgs)
	}
	if err := validateArgs(args); err != nil {
		return nil, err
	}

	return &BinPacking{
		Args: args,
		rfh:  rfh,
	}, nil
}

func validateArgs(args *config.BinPackingArgs) error {
	if len(args.ResourceItems) > 0 {
		var weightTotal int64 = 0
		for _, item := range args.ResourceItems {
			weightTotal += item.Weight
		}
		if weightTotal == 0 {
			return fmt.Errorf("unexpected BinPackingArgs: weight sum is zero")
		}
	}
	return nil
}

func (bp *BinPacking) Name() string {
	return Name
}

func (bp *BinPacking) Detect(_ context.Context) (api.DetectorResult, error) {
	var res api.DetectorResult
	nodeList := bp.rfh.NodeInfosInScope()
	for _, node := range nodeList {
		resourceUtilization := computeResourceUtilization(node, bp.Args.ResourceItems)
		if reachThresholds(resourceUtilization, bp.Args.ThresholdPercentage) {
			res.Nodes = append(res.Nodes, framework.NodeStatus{
				Name: node.GetNodeName(),
			})
		}
	}
	// then return result
	return res, nil
}

func (bp *BinPacking) SyncPolicyStore(ctx context.Context, snapshot *reschedulercache.ReschedulingSnapshot) policies.PolicyStore {
	return newBinPackingStore(snapshot, bp.Args.ResourceItems)
}

func (bp *BinPacking) Reschedule(ctx context.Context, detectRes api.DetectorResult) (api.RescheduleResult, error) {
	var res api.RescheduleResult
	for _, nodeStatus := range detectRes.Nodes {
		node, err := bp.rfh.GetNodeInfo(nodeStatus.Name)
		if err != nil {
			return api.RescheduleResult{}, fmt.Errorf("Failed to get node %s from snapshot: %v", nodeStatus.Name, err)
		}
		var pods []*v1.Pod
		if len(nodeStatus.Pods) > 0 {
			pods = nodeStatus.Pods
		} else {
			for _, pInfo := range node.GetPods() {
				pods = append(pods, pInfo.Pod)
			}
		}
		for _, pod := range pods {
			resourceUtilizationOrigin := computeResourceUtilization(node, bp.Args.ResourceItems)
			var resourceUtilizationMax float64 = 0
			var selectedNode string

			nodeList := bp.rfh.NodeInfosInScope()
			// need call FindNodesThatFitPod function
			podCopy := reschedulerpodutil.PrepareReschedulePod(pod)
			fitNodes, _, err := bp.rfh.FindNodesThatFitPod(ctx, podCopy, nodeList)
			if err != nil {
				klog.Infof("could not find suitable nodes for pod %s: %v", podutil.GeneratePodKey(podCopy), err)
				continue
			}
			for _, fitNode := range fitNodes {
				fitNodeName := fitNode.GetNodeName()
				if fitNodeName == nodeStatus.Name {
					continue
				}
				resourceUtilization := computeResourceUtilization(fitNode, bp.Args.ResourceItems)
				if resourceUtilization < resourceUtilizationOrigin {
					continue
				}
				if resourceUtilization > float64(resourceUtilizationMax) {
					resourceUtilizationMax = resourceUtilization
					selectedNode = fitNodeName
				}
			}
			if len(selectedNode) == 0 {
				continue
			}
			if _, res := bp.rfh.CheckMovementItem(nodeStatus.Name, selectedNode, pod, cache.AlgorithmStore); res != policies.Pass {
				continue
			}

			if err = bp.rfh.RemovePod(pod, cache.AlgorithmStore); err != nil {
				return api.RescheduleResult{}, fmt.Errorf("Failed to remove pod %s/%s from node %s: %v", pod.Namespace, pod.Name, nodeStatus.Name, err)
			}
			if err = bp.rfh.AssumePod(podCopy, selectedNode, cache.AlgorithmStore); err != nil {
				return api.RescheduleResult{}, fmt.Errorf("Failed to assume pod %s/%s to node %s: %v", podCopy.Namespace, podCopy.Name, selectedNode, err)
			}
			res.PodMovements = append(res.PodMovements, framework.PodMovement{
				PreviousNode: nodeStatus.Name,
				Pod:          pod,
				TargetNode:   selectedNode,
			})
		}
	}
	return res, nil
}

func computeResourceUtilization(node godel_framework.NodeInfo, thresholds []config.ResourceItem) float64 {
	allocatable := node.GetGuaranteedAllocatable()
	requested := node.GetGuaranteedRequested()
	var (
		resourceRequested   int64
		resourceAllocatable int64
		utilizationTotal    float64
		weightTotal         int64
	)
	for _, item := range thresholds {
		switch item.Resource {
		case v1.ResourceCPU.String():
			resourceRequested = requested.MilliCPU
			resourceAllocatable = allocatable.MilliCPU
		case v1.ResourceMemory.String():
			resourceRequested = requested.Memory
			resourceAllocatable = allocatable.Memory
		default:
			resourceRequested = requested.ScalarResources[v1.ResourceName(item.Resource)]
			resourceAllocatable = allocatable.ScalarResources[v1.ResourceName(item.Resource)]
		}
		utilizationTotal += float64(item.Weight) * (float64(resourceRequested) / float64(resourceAllocatable))
		weightTotal += item.Weight
	}
	return utilizationTotal / float64(weightTotal)
}

func reachThresholds(resourceUtilization float64, threshold float64) bool {
	return resourceUtilization <= threshold
}
