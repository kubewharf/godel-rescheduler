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

package runtime

import (
	"fmt"
	"time"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/framework/api/config"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	schedulerframework "github.com/kubewharf/godel-scheduler/pkg/scheduler/framework"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/runtime"
	schedulerutil "github.com/kubewharf/godel-scheduler/pkg/scheduler/util"
	"github.com/kubewharf/godel-scheduler/pkg/util/constraints"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
)

// podScheduler is the component managing cache such as node and pod info, and other configs sharing the same life cycle with scheduler
type PodScheduler struct {
	// pluginRegistry is the collection of all enabled plugins
	pluginRegistry framework.PluginMap
	// pluginOrder is the ordered list of all filter plugins
	pluginOrder framework.PluginOrder

	// basePlugins is the collection of all plugins supposed to run when a pod is scheduled
	// basePlugins are supposed to run always
	basePlugins framework.PluginCollectionSet

	metricsRecorder *runtime.MetricsRecorder
}

func NewPodScheduler(
	pluginRegistry framework.PluginMap,
	basePlugins framework.PluginCollectionSet,
	pluginArgs map[string]*schedulerconfig.PluginConfig,
) *PodScheduler {
	gs := &PodScheduler{
		basePlugins:     basePlugins,
		metricsRecorder: runtime.NewMetricsRecorder(1000, time.Second, 0, "", ""),
		pluginRegistry:  pluginRegistry,
	}

	orderedPluginRegistry := schedulerframework.NewOrderedPluginRegistry()
	gs.pluginOrder = schedulerutil.GetListIndex(orderedPluginRegistry)

	return gs
}

func (gs *PodScheduler) GetFrameworkForPod(pod *v1.Pod) (framework.SchedulerFramework, error) {
	podKey := podutil.GetPodKey(pod)
	hardConstraints, err := gs.retrievePluginsFromPodConstraints(pod, constraints.HardConstraintsAnnotationKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get hard constraints for pod %v: %v", podKey, err)
	}

	return runtime.NewPodFramework(gs.pluginRegistry, gs.pluginOrder, gs.getBasePluginsForPod(pod), hardConstraints, &framework.PluginCollection{}, gs.metricsRecorder)
}

// retrievePluginsFromPodConstraints provides constraints to run for each pod.
// Try to use constraints provided by the pod. If pod does not specify any constraints, use default ones from defaultConstraints
func (gs *PodScheduler) retrievePluginsFromPodConstraints(pod *v1.Pod, constraintAnnotationKey string) (*framework.PluginCollection, error) {
	podConstraints, err := config.GetConstraints(pod, constraintAnnotationKey)
	if err != nil {
		return nil, err
	}
	size := len(podConstraints)
	specs := make([]*framework.PluginSpec, size)
	for index, constraint := range podConstraints {
		specs[index] = framework.NewPluginSpecWithWeight(constraint.PluginName, constraint.Weight)
	}
	switch constraintAnnotationKey {
	case constraints.HardConstraintsAnnotationKey:
		return &framework.PluginCollection{
			Filters: specs,
		}, nil
	case constraints.SoftConstraintsAnnotationKey:
		return &framework.PluginCollection{
			Scores: specs,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported constraintType %v", constraintAnnotationKey)
	}
}

// getBasePluginsForPod lists the default plugins for each pod
func (gs *PodScheduler) getBasePluginsForPod(pod *v1.Pod) *framework.PluginCollection {
	podLauncher, err := podutil.GetPodLauncher(pod)
	if err != nil {
		return nil
	}
	return gs.basePlugins[string(podLauncher)]
}
