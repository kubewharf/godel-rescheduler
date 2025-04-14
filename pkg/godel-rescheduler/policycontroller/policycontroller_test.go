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

package policycontroller

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/runtime"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policyconfig"
	reschedulertesting "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/testing"
	cmdutil "github.com/kubewharf/godel-rescheduler/pkg/util/cmd"

	godelclientfake "github.com/kubewharf/godel-scheduler-api/pkg/client/clientset/versioned/fake"
	crdinformers "github.com/kubewharf/godel-scheduler-api/pkg/client/informers/externalversions"
	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	godel_framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	pdbstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/pdb_store"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
)

type StoreA struct{}

var (
	_ policies.PolicyStore = &StoreA{}
)

func (storeA *StoreA) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod) policies.CheckResult {
	return policies.Pass
}

func (store *StoreA) AssumePod(pod *v1.Pod) {}

func (store *StoreA) RemovePod(pod *v1.Pod) {}

type PluginA struct{}

var (
	_ api.DetectorPlugin          = &PluginA{}
	_ api.AlgorithmPlugin         = &PluginA{}
	_ api.PolicyStoreSyncerPlugin = &PluginA{}
)

func NewPluginA(plArgs apiruntime.Object, rfh api.ReschedulerFrameworkHandle) (godel_framework.Plugin, error) {
	return &PluginA{}, nil
}

func (pluginA *PluginA) Name() string {
	return "PluginA"
}

func (pluginA *PluginA) Detect(_ context.Context) (api.DetectorResult, error) {
	return api.DetectorResult{
		Nodes: []api.NodeStatus{
			{
				Name: "node",
				Pods: []*v1.Pod{
					makePod("foo", "node"),
				},
			},
		},
	}, nil
}

func (pluginA *PluginA) SyncPolicyStore(ctx context.Context, snapshot *cache.ReschedulingSnapshot) policies.PolicyStore {
	return &StoreA{}
}

func (pluginA *PluginA) Reschedule(ctx context.Context, detectRes api.DetectorResult) (api.RescheduleResult, error) {
	return api.RescheduleResult{
		PodMovements: []api.PodMovement{
			{
				PreviousNode: detectRes.Nodes[0].Name,
				TargetNode:   "new_node",
				Pod:          detectRes.Nodes[0].Pods[0],
			},
		},
	}, nil
}

type StoreB struct{}

var (
	_ policies.PolicyStore = &StoreB{}
)

func (storeB *StoreB) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod) policies.CheckResult {
	return policies.MovedInstanceError
}

func (store *StoreB) AssumePod(pod *v1.Pod) {}

func (store *StoreB) RemovePod(pod *v1.Pod) {}

type PluginB struct{}

var (
	_ api.DetectorPlugin          = &PluginB{}
	_ api.AlgorithmPlugin         = &PluginB{}
	_ api.PolicyStoreSyncerPlugin = &PluginB{}
)

func NewPluginB(plArgs apiruntime.Object, rfh api.ReschedulerFrameworkHandle) (godel_framework.Plugin, error) {
	return &PluginB{}, nil
}

func (pluginB *PluginB) Name() string {
	return "PluginB"
}

func (pluginB *PluginB) Detect(_ context.Context) (api.DetectorResult, error) {
	return api.DetectorResult{
		Nodes: []api.NodeStatus{
			{
				Name: "node",
				Pods: []*v1.Pod{
					makePod("foo", "node"),
				},
			},
		},
	}, nil
}

func (pluginB *PluginB) SyncPolicyStore(ctx context.Context, snapshot *cache.ReschedulingSnapshot) policies.PolicyStore {
	return &StoreB{}
}

func (pluginB *PluginB) Reschedule(ctx context.Context, detectRes api.DetectorResult) (api.RescheduleResult, error) {
	return api.RescheduleResult{
		PodMovements: []api.PodMovement{
			{
				PreviousNode: detectRes.Nodes[0].Name,
				TargetNode:   "new_node",
				Pod:          detectRes.Nodes[0].Pods[0],
			},
		},
	}, nil
}

type StoreC struct{}

var (
	_ policies.PolicyStore = &StoreC{}
)

func (storeC *StoreC) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod) policies.CheckResult {
	return policies.Pass
}

func (store *StoreC) AssumePod(pod *v1.Pod) {}

func (store *StoreC) RemovePod(pod *v1.Pod) {}

type PluginC struct{}

var (
	_ api.DetectorPlugin          = &PluginC{}
	_ api.AlgorithmPlugin         = &PluginC{}
	_ api.PolicyStoreSyncerPlugin = &PluginC{}
)

func NewPluginC(plArgs apiruntime.Object, rfh api.ReschedulerFrameworkHandle) (godel_framework.Plugin, error) {
	return &PluginC{}, nil
}

func (pluginC *PluginC) Name() string {
	return "PluginC"
}

func (pluginC *PluginC) Detect(_ context.Context) (api.DetectorResult, error) {
	return api.DetectorResult{
		Nodes: []api.NodeStatus{
			{
				Name: "node",
				Pods: []*v1.Pod{
					makePod("foo", "node"),
				},
			},
		},
	}, nil
}

func (pluginC *PluginC) SyncPolicyStore(ctx context.Context, snapshot *cache.ReschedulingSnapshot) policies.PolicyStore {
	return &StoreC{}
}

func (pluginC *PluginC) Reschedule(ctx context.Context, detectRes api.DetectorResult) (api.RescheduleResult, error) {
	return api.RescheduleResult{
		PodMovements: []api.PodMovement{
			{
				PreviousNode: detectRes.Nodes[0].Name,
				TargetNode:   "new_node",
				Pod:          detectRes.Nodes[0].Pods[0],
			},
		},
	}, nil
}

func makePod(name, node string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(name),
		},
		Spec: v1.PodSpec{
			NodeName: node,
		},
	}
}

func makeNode(name string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func TestCheckMovementItems(t *testing.T) {
	// one check succeed, one check fail
	{
		var mu sync.Mutex
		stop := make(chan struct{})
		findNodesThatFitPod := func(ctx context.Context, pod *v1.Pod, nodes []godel_framework.NodeInfo) ([]godel_framework.NodeInfo, godel_framework.NodeToStatusMap, error) {
			node := &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new_node",
				},
			}
			nodeInfo := godel_framework.NewNodeInfo()
			nodeInfo.SetNode(node)
			return []godel_framework.NodeInfo{nodeInfo}, nil, nil
		}
		handle := reschedulertesting.NewMockReschedulerFrameworkHandle(findNodesThatFitPod, nil, nil, nil)

		crdClient := godelclientfake.NewSimpleClientset()
		crdInformerFactory := crdinformers.NewSharedInformerFactory(crdClient, 0)
		client := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(client, 0)
		handlerWrapper := commoncache.MakeCacheHandlerWrapper().
			ComponentName(schedulerconfig.DefaultGodelSchedulerName).SchedulerType(schedulerconfig.DefaultSchedulerName).SubCluster(godel_framework.DefaultSubCluster).
			PodAssumedTTL(15 * time.Minute).Period(10 * time.Second).StopCh(stop).
			EnableStore(string(pdbstore.Name))
		reschedulerCache := cache.New(handlerWrapper.Obj())

		node1 := makeNode("node")
		node2 := makeNode("new_node")
		pod := makePod("foo", "node")
		reschedulerCache.AddNode(node1)
		reschedulerCache.AddNode(node2)
		reschedulerCache.AddPod(pod)
		snapshotLister := cache.NewReschedulingSnapshotLister(5, informerFactory, crdInformerFactory)
		duration := 2 * time.Second
		pluginA, _ := NewPluginA(nil, handle)
		pluginB, _ := NewPluginB(nil, handle)
		pluginC, _ := NewPluginC(nil, handle)
		policyConfig := &policyconfig.PolicyConfig{
			Detector: policyconfig.Detector{
				DetectorPlugin: pluginA.(api.DetectorPlugin),
				DetectorTrigger: &api.PolicyTrigger{
					Period: &duration,
				},
			},
			Algorithm: pluginA.(api.AlgorithmPlugin),
			CheckPoliciesForAlgorithm: []api.PolicyStoreSyncerPlugin{
				pluginB.(api.PolicyStoreSyncerPlugin),
				pluginC.(api.PolicyStoreSyncerPlugin),
			},
			Scope: map[string]string{
				"nodeLevel": "default",
			},
		}
		metricsRecorder := runtime.NewMetricsRecorder(1000, time.Second)
		eventBroadcaster := cmdutil.NewEventBroadcasterAdapter(client)
		eventRecorder := eventBroadcaster.NewRecorder("rescheduler")
		movementLister := crdInformerFactory.Scheduling().V1alpha1().Movements().Lister()
		schedulerLister := crdInformerFactory.Scheduling().V1alpha1().Schedulers().Lister()
		podLister := informerFactory.Core().V1().Pods().Lister()
		var generation uint = 0
		policyController := New(reschedulerCache, make(chan map[string]api.PodMovement), &mu, policyConfig, snapshotLister, metricsRecorder, eventRecorder, handle, &generation, true, crdClient, client, schedulerLister, movementLister, podLister)

		movements := []string{}
		go policyController.run(stop, &movements)
		time.Sleep(3 * time.Second)
		if len(movements) > 0 {
			t.Errorf("expected failed rescheduling")
		}
		close(stop)
	}

	// all check succeed
	{
		var mu sync.Mutex
		stop := make(chan struct{})
		findNodesThatFitPod := func(ctx context.Context, pod *v1.Pod, nodes []godel_framework.NodeInfo) ([]godel_framework.NodeInfo, godel_framework.NodeToStatusMap, error) {
			node := &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new_node",
				},
			}
			nodeInfo := godel_framework.NewNodeInfo()
			nodeInfo.SetNode(node)
			return []godel_framework.NodeInfo{nodeInfo}, nil, nil
		}
		handle := reschedulertesting.NewMockReschedulerFrameworkHandle(findNodesThatFitPod, nil, nil, nil)

		crdClient := godelclientfake.NewSimpleClientset()
		crdInformerFactory := crdinformers.NewSharedInformerFactory(crdClient, 0)
		client := clientsetfake.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(client, 0)
		handlerWrapper := commoncache.MakeCacheHandlerWrapper().
			ComponentName(schedulerconfig.DefaultGodelSchedulerName).SchedulerType(schedulerconfig.DefaultSchedulerName).SubCluster(godel_framework.DefaultSubCluster).
			PodAssumedTTL(15 * time.Minute).Period(10 * time.Second).StopCh(stop).
			EnableStore(string(pdbstore.Name))
		reschedulerCache := cache.New(handlerWrapper.Obj())

		node1 := makeNode("node")
		node2 := makeNode("new_node")
		pod := makePod("foo", "node")
		reschedulerCache.AddNode(node1)
		reschedulerCache.AddNode(node2)
		reschedulerCache.AddPod(pod)
		snapshotLister := cache.NewReschedulingSnapshotLister(6, informerFactory, crdInformerFactory)
		duration := 10 * time.Second
		pluginA, _ := NewPluginA(nil, handle)
		pluginC, _ := NewPluginC(nil, handle)
		policyConfig := &policyconfig.PolicyConfig{
			Detector: policyconfig.Detector{
				DetectorPlugin: pluginA.(api.DetectorPlugin),
				DetectorTrigger: &api.PolicyTrigger{
					Period: &duration,
				},
			},
			Algorithm: pluginA.(api.AlgorithmPlugin),
			CheckPoliciesForAlgorithm: []api.PolicyStoreSyncerPlugin{
				pluginA.(api.PolicyStoreSyncerPlugin),
				pluginC.(api.PolicyStoreSyncerPlugin),
			},
			Scope: map[string]string{
				"nodeLevel": "default",
			},
		}
		metricsRecorder := runtime.NewMetricsRecorder(1000, time.Second)
		eventBroadcaster := cmdutil.NewEventBroadcasterAdapter(client)
		eventRecorder := eventBroadcaster.NewRecorder("rescheduler")
		movementLister := crdInformerFactory.Scheduling().V1alpha1().Movements().Lister()
		schedulerLister := crdInformerFactory.Scheduling().V1alpha1().Schedulers().Lister()
		podLister := informerFactory.Core().V1().Pods().Lister()
		var generation uint = 0
		policyController := New(reschedulerCache, make(chan map[string]api.PodMovement), &mu, policyConfig, snapshotLister, metricsRecorder, eventRecorder, handle, &generation, true, crdClient, client, schedulerLister, movementLister, podLister)

		movements := []string{}
		go policyController.run(stop, &movements)
		time.Sleep(30 * time.Second) // `movements` is expected to have a value after 10s (dectector trigger duration) + 15s (processMovement timeout threshold)
		if len(movements) == 0 {
			t.Errorf("expected successful rescheduling")
		}
		close(stop)
	}
}
