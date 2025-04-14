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

package api

import (
	"context"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	reschedulercache "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/handle"
	v1 "k8s.io/api/core/v1"
)

// ReschedulerFrameworkHandle provides data and some tools that plugins can use in Rescheduler. It is
// passed to the plugin factories at the time of plugin initialization. Plugins
// must store and use this handle to call framework functions.
type ReschedulerFrameworkHandle interface {
	// ClientSet returns a kubernetes clientSet.
	FindNodesThatFitPod(ctx context.Context, pod *v1.Pod, nodes []framework.NodeInfo) ([]framework.NodeInfo, framework.NodeToStatusMap, error)
	AssumePod(pod *v1.Pod, node string, storeType string) error
	RemovePod(pod *v1.Pod, storeType string) error
	CheckMovementItem(previousNode, targetNode string, instance *v1.Pod, storeType string) (string, policies.CheckResult)
	SetSnapshot(reschedulingSnapshot *reschedulercache.ReschedulingSnapshot)
	GeneratePluginRegistryMap()
	GetNodeInfo(nodeName string) (framework.NodeInfo, error)
	GetPod(string) (*v1.Pod, error)
	NodeInfosInScope() []framework.NodeInfo
	handle.PodFrameworkHandle
}

type DetectorPlugin interface {
	// return name of detector plugin
	Name() string
	// return result when crojob triggered or threshold triggered
	Detect(context.Context) (DetectorResult, error)
}

type AlgorithmPlugin interface {
	// return name of algorithm plugin
	Name() string
	// return newly aasigned nodes for pods
	Reschedule(context.Context, DetectorResult) (RescheduleResult, error)
}

type PolicyStoreSyncerPlugin interface {
	// return name of policy store syncer plugin
	Name() string
	// return policy store, do not change snapshot
	SyncPolicyStore(context.Context, *cache.ReschedulingSnapshot) policies.PolicyStore
}
