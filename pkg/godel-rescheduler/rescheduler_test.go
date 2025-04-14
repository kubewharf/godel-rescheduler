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
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	binpacking "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/plugins/bin-packing"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/plugins/protection"
	cmdutil "github.com/kubewharf/godel-rescheduler/pkg/util/cmd"

	schedulingv1alpha1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/scheduling/v1alpha1"
	godelclientfake "github.com/kubewharf/godel-scheduler-api/pkg/client/clientset/versioned/fake"
	crdinformers "github.com/kubewharf/godel-scheduler-api/pkg/client/informers/externalversions"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	godelpodutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	katalystclientfake "github.com/kubewharf/katalyst-api/pkg/client/clientset/versioned/fake"
	katalystinformers "github.com/kubewharf/katalyst-api/pkg/client/informers/externalversions"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
)

func TestReschedulerRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := ctx.Done()
	port := 9090

	crdClient := godelclientfake.NewSimpleClientset()
	crdInformerFactory := crdinformers.NewSharedInformerFactory(crdClient, 0)
	client := clientsetfake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(client, 0)
	katalystClient := katalystclientfake.NewSimpleClientset()
	katalystInformerFactory := katalystinformers.NewSharedInformerFactory(katalystClient, 0)

	eventBroadcaster := cmdutil.NewEventBroadcasterAdapter(client)
	eventRecorder := eventBroadcaster.NewRecorder("rescheduler")

	policies := []config.ReschedulerPolicy{
		{
			PolicyTrigger: &config.PolicyTrigger{
				Port: &port,
			},
			Detector: config.DetectorConfig{
				Name: binpacking.Name,
				Args: runtime.RawExtension{
					Object: &config.BinPackingArgs{
						ResourceItems: []config.ResourceItem{
							{
								Resource: "cpu",
								Weight:   1,
							},
						},
						ThresholdPercentage: 0.5,
					},
				},
				// DetectorTrigger: &config.PolicyTrigger{
				// 	Port: &port,
				// },
				CheckPolicies: []string{
					"Protection",
				},
			},
			Algorithm: config.AlgorithmConfig{
				Name: binpacking.Name,
				Args: runtime.RawExtension{
					Object: &config.BinPackingArgs{
						ResourceItems: []config.ResourceItem{
							{
								Resource: "cpu",
								Weight:   1,
							},
						},
						ThresholdPercentage: 0.5,
					},
				},
				CheckPolicies: []string{
					"Protection",
				},
			},
		},
		{
			Detector: config.DetectorConfig{
				Name: "Protection",
			},
			Algorithm: config.AlgorithmConfig{
				Name: "Protection",
			},
		},
	}
	resched, err := New(client, crdClient, informerFactory, crdInformerFactory, katalystInformerFactory, stop, eventRecorder, schedulerconfig.GodelSchedulerConfiguration{},
		WithReschedulerPolicyConfigs(policies))
	if err != nil {
		t.Errorf("failed to new rescheduler: %v", err)
	}

	// Start all informers.
	informerFactory.Start(ctx.Done())
	crdInformerFactory.Start(ctx.Done())

	// Wait for all caches to sync before scheduling.
	informerFactory.WaitForCacheSync(ctx.Done())
	crdInformerFactory.WaitForCacheSync(ctx.Done())
	go resched.Run(ctx)
	time.Sleep(1 * time.Second)

	n1 := testing_helper.MakeNode().Name("n1").Capacity(map[v1.ResourceName]string{"cpu": "4"}).Obj()
	n2 := testing_helper.MakeNode().Name("n2").Capacity(map[v1.ResourceName]string{"cpu": "4"}).Obj()
	client.CoreV1().Nodes().Create(ctx, n1, metav1.CreateOptions{})
	client.CoreV1().Nodes().Create(ctx, n2, metav1.CreateOptions{})

	pdb := testing_helper.MakePdb().Namespace("default").Name("pdb").DisruptionsAllowed(0).
		Selector(&metav1.LabelSelector{MatchLabels: map[string]string{"k": "v"}}).Obj()
	client.PolicyV1().PodDisruptionBudgets(pdb.Namespace).Create(ctx, pdb, metav1.CreateOptions{})

	req := map[v1.ResourceName]string{"cpu": "1"}
	ref := metav1.OwnerReference{
		Kind: godelpodutil.ReplicaSetKind,
		Name: "rs",
		UID:  "rs",
	}

	tests := []struct {
		name             string
		addPods          []*v1.Pod
		updatePods       []*v1.Pod
		expectedMovement *schedulingv1alpha1.Movement
	}{
		{
			name: "violating pdb, generated movements failed",
			addPods: []*v1.Pod{
				testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p4").UID("p4").Node(n2.Name).Req(req).ControllerRef(ref).Label("k", "v").Obj(),
			},
		},
		{
			name: "not pass check policies",
			updatePods: []*v1.Pod{
				testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p4").UID("p4").Node(n2.Name).Req(req).ControllerRef(ref).
					Annotation(protection.ExcludeFromReschedulingAnnotationKey, protection.ExcludeFromReschedulingTrue).Obj(),
			},
		},
		{
			name: "success",
			updatePods: []*v1.Pod{
				testing_helper.MakePod().Namespace("default").Name("p1").UID("p1").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p2").UID("p2").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p3").UID("p3").Node(n1.Name).Req(req).ControllerRef(ref).Obj(),
				testing_helper.MakePod().Namespace("default").Name("p4").UID("p4").Node(n2.Name).Req(req).ControllerRef(ref).Obj(),
			},
			expectedMovement: &schedulingv1alpha1.Movement{
				Spec: schedulingv1alpha1.MovementSpec{
					DeletedTasks: []*schedulingv1alpha1.TaskInfo{
						{
							Name:      "p4",
							Namespace: "default",
							UID:       "p4",
							Node:      n2.Name,
						},
					},
					Creator:    binpacking.Name,
					Generation: 0,
				},
				Status: schedulingv1alpha1.MovementStatus{
					Owners: []*schedulingv1alpha1.Owner{
						{
							Owner: &schedulingv1alpha1.OwnerInfo{
								Type:      godelpodutil.ReplicaSetKind,
								Name:      "rs",
								Namespace: "default",
								UID:       "rs",
							},
							RecommendedNodes: []*schedulingv1alpha1.RecommendedNode{
								{
									Node:            n1.Name,
									DesiredPodCount: 1,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, p := range tt.updatePods {
				client.CoreV1().Pods(p.Namespace).Update(ctx, p, metav1.UpdateOptions{})
			}
			for _, p := range tt.addPods {
				client.CoreV1().Pods(p.Namespace).Create(ctx, p, metav1.CreateOptions{})
			}

			url := fmt.Sprintf("http://127.0.0.1:%d/trigger", port)
			resp, err := http.Get(url)
			if err != nil {
				t.Errorf("fail to trigger: %v", err)
			} else if resp.StatusCode != 200 {
				t.Errorf("fail to trigger: %v", resp)
			}
			time.Sleep(time.Second)
			movements, err := crdInformerFactory.Scheduling().V1alpha1().Movements().Lister().List(labels.Everything())
			if err != nil {
				t.Errorf("failed to list movements: %v", err)
			}
			if tt.expectedMovement != nil {
				if len(movements) != 1 {
					t.Errorf("expected 1 movement, byt got %d", len(movements))
				} else {
					if !reflect.DeepEqual(tt.expectedMovement.Spec, movements[0].Spec) {
						t.Errorf("expected movement spec: %v, but got: %v", tt.expectedMovement.Spec, movements[0].Spec)
					}
					if !reflect.DeepEqual(tt.expectedMovement.Status, movements[0].Status) {
						t.Errorf("expected movement status: %v, but got: %v", tt.expectedMovement.Status, movements[0].Status)
					}
				}
			} else {
				if len(movements) > 0 {
					t.Errorf("expected nil movement, but got %d", len(movements))
				}
			}

		})
	}
	time.Sleep(1 * time.Second)
}
