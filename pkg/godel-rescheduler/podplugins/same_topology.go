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

package podplugins

import (
	"context"

	reschedulerpodutil "github.com/kubewharf/godel-rescheduler/pkg/util/pod"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	podgroupstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/podgroup_store"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/handle"
	"github.com/kubewharf/godel-scheduler/pkg/util/helper"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	unitutil "github.com/kubewharf/godel-scheduler/pkg/util/unit"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	SameTopologyPluginName = "SameTopology"
	preFilterStateKey      = "PreFilter" + SameTopologyPluginName
)

// preFilterState computed at PreFilter and used at Filter.
type preFilterState struct {
	podLauncher          podutil.PodLauncher
	topologyRequiredKVs  map[string]string
	topologyPreferredKVs map[string]string
	nodeSelector         *v1.NodeSelector
}

// Clone the prefilter state.
func (s *preFilterState) Clone() framework.StateData {
	return s
}

// sameTopology is a plugin that checks if a node satisfy the same topology.
type sameTopology struct {
	handler handle.PodFrameworkHandle
}

var (
	_ framework.FilterPlugin = &sameTopology{}
	_ framework.StateData    = &preFilterState{}
)

func NewSamaTopologyPlugin(_ runtime.Object, handle handle.PodFrameworkHandle) (framework.Plugin, error) {
	return &sameTopology{
		handler: handle,
	}, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (s *sameTopology) Name() string {
	return SameTopologyPluginName
}

// PreFilter invoked at the prefilter extension point.
func (s *sameTopology) PreFilter(_ context.Context, cycleState *framework.CycleState, pod *v1.Pod) *framework.Status {
	pgName := unitutil.GetPodGroupFullName(pod)
	if len(pgName) == 0 {
		return nil
	}
	// TODO: remove the GetPodLauncher if we can bootstrap for pod.
	podLauncher, err := podutil.GetPodLauncher(pod)
	if err != nil {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, err.Error())
	}
	var pluginHandle podgroupstore.StoreHandle
	if ins := s.handler.FindStore(podgroupstore.Name); ins != nil {
		pluginHandle = ins.(podgroupstore.StoreHandle)
	}
	podGroup, err := pluginHandle.GetPodGroupInfo(pgName)
	if err != nil {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, err.Error())
	}
	if podGroup != nil && podGroup.Spec.Affinity != nil && podGroup.Spec.Affinity.PodGroupAffinity != nil {
		requiredAffinity := podGroup.Spec.Affinity.PodGroupAffinity.Required
		preferredAffinity := podGroup.Spec.Affinity.PodGroupAffinity.Preferred

		if len(requiredAffinity) > 0 || len(preferredAffinity) > 0 {
			oldNodeInfo, err := s.handler.SnapshotSharedLister().NodeInfos().Get(reschedulerpodutil.GetPodNodeBeforeReschedule(pod))
			if err != nil {
				return framework.NewStatus(framework.UnschedulableAndUnresolvable, err.Error())
			}

			topologyRequiredKVs := make(map[string]string)
			topologyPreferredKVs := make(map[string]string)
			labels := oldNodeInfo.GetNodeLabels(podLauncher)
			// TODO: Currently, we can only migrate within one nodegroup.
			// If the entire PodGroup-Pods are rescheduled (how to judge?), There is no need for such restrictions.
			for i := range requiredAffinity {
				key := requiredAffinity[i].TopologyKey
				value, ok := labels[key]
				if !ok {
					return framework.NewStatus(framework.UnschedulableAndUnresolvable, "Mismatch for required topology key")
				}
				topologyRequiredKVs[key] = value
			}

			for i := range preferredAffinity {
				key := preferredAffinity[i].TopologyKey
				value, ok := labels[key]
				if !ok {
					return framework.NewStatus(framework.UnschedulableAndUnresolvable, "Mismatch for preferred topology key")
				}
				topologyPreferredKVs[key] = value
			}
			cycleState.Write(preFilterStateKey, &preFilterState{podLauncher, topologyRequiredKVs, topologyPreferredKVs, podGroup.Spec.Affinity.PodGroupAffinity.NodeSelector})
		}
	}

	return framework.NewStatus(framework.Success)
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
func (s *sameTopology) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (s *sameTopology) Filter(_ context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodeInfo framework.NodeInfo) *framework.Status {
	stateObj, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		// If the pod doesn't have any same affinity, pass directly.
		return framework.NewStatus(framework.Success)
	}
	state, ok := stateObj.(*preFilterState)
	if !ok {
		return framework.NewStatus(framework.Error, "SameTopology.preFilterState convert error")
	}

	labels := nodeInfo.GetNodeLabels(state.podLauncher)

	for key, value := range state.topologyRequiredKVs {
		if v, ok := labels[key]; !ok || v != value {
			return framework.NewStatus(framework.UnschedulableAndUnresolvable, "Mismatch for required topology key")
		}
	}

	for key, value := range state.topologyPreferredKVs {
		if v, ok := labels[key]; !ok || v != value {
			return framework.NewStatus(framework.UnschedulableAndUnresolvable, "Mismatch for preferred topology key")
		}
	}

	if state.nodeSelector != nil {
		if !helper.MatchNodeSelectorTerms(state.nodeSelector.NodeSelectorTerms, nodeInfo.GetNodeLabels(state.podLauncher), nil) {
			return framework.NewStatus(framework.UnschedulableAndUnresolvable, "Mismatch for podgroup node selector")
		}
	}

	return framework.NewStatus(framework.Success)
}
