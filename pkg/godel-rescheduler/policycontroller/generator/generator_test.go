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

package generator

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/checker"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/internal"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/relation"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/testdata"
	"github.com/kubewharf/godel-rescheduler/pkg/util"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	godelutil "github.com/kubewharf/godel-scheduler/pkg/util"
	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
)

func TestGenerator(t *testing.T) {
	// Test CheckFunc, can not select more than four edges.
	testCheckFunc := func(_ internal.Graph, es internal.EdgeSet, e ...internal.Edge) error {
		if es.Len()+len(e) <= 4 {
			return nil
		}
		return fmt.Errorf("can not select more than four edges")
	}
	specialJudgeFunc := func(relations relation.Relations) error {
		if len(relations) <= 4 {
			return nil
		}
		return fmt.Errorf("error occurred, expect len(relation)<=4, but got %v", len(relations))
	}

	type args struct {
		ctx       context.Context
		relations relation.Relations
		checkFunc internal.CheckFunc
	}
	tests := []struct {
		name             string
		args             args
		specialJudgeFunc func(relation.Relations) error
	}{
		{
			name: "graph 0",
			args: args{
				ctx:       context.Background(),
				relations: testdata.SelectTestData(0).Relations,
				checkFunc: testCheckFunc,
			},
			specialJudgeFunc: specialJudgeFunc,
		},
		{
			name: "graph 1",
			args: args{
				ctx:       context.Background(),
				relations: testdata.SelectTestData(1).Relations,
				checkFunc: testCheckFunc,
			},
			specialJudgeFunc: specialJudgeFunc,
		},
		{
			name: "graph 2",
			args: args{
				ctx:       context.Background(),
				relations: testdata.SelectTestData(2).Relations,
				checkFunc: testCheckFunc,
			},
			specialJudgeFunc: specialJudgeFunc,
		},
		{
			name: "graph 3",
			args: args{
				ctx:       context.Background(),
				relations: testdata.SelectTestData(3).Relations,
				checkFunc: testCheckFunc,
			},
			specialJudgeFunc: specialJudgeFunc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.args.ctx, tt.args.relations, tt.args.checkFunc)

			for !generator.Finished() {
				relations, err := generator.Generate(tt.args.ctx)
				if err != nil {
					t.Errorf("Error occurred when Generate: %v", err)
				}
				if err := tt.specialJudgeFunc(relations); err != nil {
					t.Errorf("Could not pass special judge: %v", err)
				}
			}
		})
	}
}

// TODO: move this to test-helper
type fakePdbItem struct {
	pdb            *policy.PodDisruptionBudget
	pdbSelector    labels.Selector
	replicaSetKeys framework.GenerationStringSet
	daemonSetKeys  framework.GenerationStringSet
	generation     int64
}

func (p *fakePdbItem) GetPDB() *policy.PodDisruptionBudget {
	return p.pdb
}

func (p *fakePdbItem) SetPDB(pdb *policy.PodDisruptionBudget) {
	p.pdb = pdb
}

func (p *fakePdbItem) GetPDBSelector() labels.Selector {
	return p.pdbSelector
}

func (p *fakePdbItem) SetPDBSelector(selector labels.Selector) {
	p.pdbSelector = selector
}

func (p *fakePdbItem) RemoveAllOwners() {
	p.replicaSetKeys = framework.NewGenerationStringSet()
	p.daemonSetKeys = framework.NewGenerationStringSet()
}

func (p *fakePdbItem) AddOwner(ownerType, key string) {
	switch ownerType {
	case godelutil.OwnerTypeReplicaSet:
		p.replicaSetKeys.Insert(key)
	case godelutil.OwnerTypeDaemonSet:
		p.daemonSetKeys.Insert(key)
	default:
		return
	}
}

func (p *fakePdbItem) RemoveOwner(ownerType, key string) {
	switch ownerType {
	case godelutil.OwnerTypeReplicaSet:
		p.replicaSetKeys.Delete(key)
	case godelutil.OwnerTypeDaemonSet:
		p.daemonSetKeys.Delete(key)
	default:
		return
	}
}

func (p *fakePdbItem) GetRelatedOwnersByType(ownerType string) []string {
	switch ownerType {
	case godelutil.OwnerTypeReplicaSet:
		return p.replicaSetKeys.Strings()
	case godelutil.OwnerTypeDaemonSet:
		return p.daemonSetKeys.Strings()
	}
	return nil
}

func (p *fakePdbItem) RemovePDBFromOwner(removeFunc func(string, string)) {
	p.replicaSetKeys.Range(func(rs string) {
		removeFunc(godelutil.OwnerTypeReplicaSet, rs)
	})
	p.daemonSetKeys.Range(func(ds string) {
		removeFunc(godelutil.OwnerTypeDaemonSet, ds)
	})
}

func (p *fakePdbItem) GetGeneration() int64 {
	return 0
}

func (p *fakePdbItem) SetGeneration(generation int64) {
	return
}

func (p *fakePdbItem) Replace(item framework.PDBItem) {
	return
}

func (p *fakePdbItem) Clone() framework.PDBItem {
	return &fakePdbItem{
		pdb:         p.pdb.DeepCopy(),
		pdbSelector: p.pdbSelector.DeepCopySelector(),
	}
}

var defaultNamespace = "default"

func makePod(dp, name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultNamespace,
			Name:      name,
			UID:       types.UID(dp + "/" + name),
			Labels: map[string]string{
				"dp": dp,
			},
		},
	}
}

func TestGeneratorPDBChecker(t *testing.T) {
	specialJudgeFunc := func(pods map[string]*v1.Pod, pdbItems []framework.PDBItem, relations relation.Relations) error {
		pdbsAllowed := make([]int32, len(pdbItems))
		for i, pdbItem := range pdbItems {
			pdb := pdbItem.GetPDB()
			pdbsAllowed[i] = pdb.Status.DisruptionsAllowed
		}

		for _, r := range relations {
			pod := pods[r.GetName()]

			for i, pdbItem := range pdbItems {
				pdb := pdbItem.GetPDB()
				if pdb.Namespace != pod.Namespace {
					continue
				}
				selector := pdbItem.GetPDBSelector()
				if selector == nil || selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
					continue
				}

				if _, exist := pdb.Status.DisruptedPods[pod.Name]; exist {
					continue
				}
				pdbsAllowed[i]--
				if pdbsAllowed[i] < 0 {
					return fmt.Errorf("pod %s violating pdb %s(%d)", r.GetName(), util.GetPDBKey(pdb), pdb.Status.DisruptionsAllowed)
				}
			}
		}
		return nil
	}

	type args struct {
		ctx       context.Context
		relations relation.Relations
		pods      map[string]*v1.Pod
		pdbItems  []framework.PDBItem
	}
	requirement, _ := labels.NewRequirement("dp", selection.In, []string{"dp1", "dp2", "dp3", "dp4"})

	tests := []struct {
		name             string
		args             args
		want             []relation.Relations
		specialJudgeFunc func(map[string]*v1.Pod, []framework.PDBItem, relation.Relations) error
	}{
		{
			name: "graph 0",
			args: args{
				ctx:       context.Background(),
				relations: testdata.SelectTestData(0).Relations,
				pods: map[string]*v1.Pod{
					"dp1/pod2": makePod("dp1", "pod2"),
					"dp1/pod1": makePod("dp1", "pod1"),
				},
				pdbItems: []framework.PDBItem{
					&fakePdbItem{
						pdb: &policy.PodDisruptionBudget{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: defaultNamespace,
							},
							Status: policy.PodDisruptionBudgetStatus{
								DisruptionsAllowed: 1,
							},
						},
						pdbSelector: labels.NewSelector().Add(*requirement),
					},
				},
			},
			specialJudgeFunc: specialJudgeFunc,
		},
		{
			name: "graph 1",
			args: args{
				ctx:       context.Background(),
				relations: testdata.SelectTestData(1).Relations,
				pods: map[string]*v1.Pod{
					"dp1/pod1": makePod("dp1", "pod1"),
					"dp1/pod2": makePod("dp1", "pod2"),
					"dp1/pod3": makePod("dp1", "pod3"),
					"dp1/pod4": makePod("dp1", "pod4"),
					"dp1/pod5": makePod("dp1", "pod5"),
					"dp2/pod1": makePod("dp2", "pod1"),
					"dp2/pod2": makePod("dp2", "pod2"),
					"dp2/pod3": makePod("dp2", "pod3"),
					"dp2/pod4": makePod("dp2", "pod4"),
					"dp3/pod1": makePod("dp3", "pod1"),
					"dp3/pod2": makePod("dp3", "pod2"),
				},
				pdbItems: []framework.PDBItem{
					&fakePdbItem{
						pdb: &policy.PodDisruptionBudget{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: defaultNamespace,
							},
							Status: policy.PodDisruptionBudgetStatus{
								DisruptionsAllowed: 4,
							},
						},
						pdbSelector: labels.NewSelector().Add(*requirement),
					},
				},
			},
			specialJudgeFunc: specialJudgeFunc,
		},
		{
			name: "graph 2",
			args: args{
				ctx:       context.Background(),
				relations: testdata.SelectTestData(2).Relations,
				pods: map[string]*v1.Pod{
					"dp1/pod1": makePod("dp1", "pod1"),
					"dp1/pod2": makePod("dp1", "pod2"),
					"dp2/pod1": makePod("dp2", "pod1"),
					"dp2/pod2": makePod("dp2", "pod2"),
					"dp2/pod3": makePod("dp2", "pod3"),
					"dp2/pod4": makePod("dp2", "pod4"),
					"dp3/pod1": makePod("dp3", "pod1"),
					"dp3/pod2": makePod("dp3", "pod2"),
					"dp3/pod3": makePod("dp3", "pod3"),
					"dp4/pod1": makePod("dp4", "pod1"),
					"dp4/pod2": makePod("dp4", "pod2"),
				},
				pdbItems: []framework.PDBItem{
					&fakePdbItem{
						pdb: &policy.PodDisruptionBudget{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: defaultNamespace,
							},
							Status: policy.PodDisruptionBudgetStatus{
								DisruptionsAllowed: 2,
							},
						},
						pdbSelector: labels.NewSelector().Add(*requirement),
					},
				},
			},
			specialJudgeFunc: specialJudgeFunc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := checker.NewPDBChecker()
			generator := NewGenerator(tt.args.ctx, tt.args.relations, checker.GetCheckFunc())

			for !generator.Finished() {
				checker.Refresh(tt.args.pods, tt.args.pdbItems)
				relations, err := generator.Generate(tt.args.ctx)
				if err != nil {
					t.Errorf("Error occurred when Generate: %v", err)
				}
				if err := tt.specialJudgeFunc(tt.args.pods, tt.args.pdbItems, relations); err != nil {
					t.Errorf("Could not pass special judge: %v", err)
				}
			}
		})
	}
}
