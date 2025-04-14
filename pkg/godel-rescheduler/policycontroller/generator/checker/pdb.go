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

package checker

import (
	"fmt"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/internal"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/util"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type PDBChecker struct {
	pods     map[string]*v1.Pod
	pdbItems []framework.PDBItem
}

func NewPDBChecker() *PDBChecker {
	return &PDBChecker{}
}

func (c *PDBChecker) Refresh(pods map[string]*v1.Pod, pdbItems []framework.PDBItem) {
	c.pods = pods
	items := make([]framework.PDBItem, len(pdbItems))
	for i := range pdbItems {
		items[i] = pdbItems[i].Clone()
	}
	c.pdbItems = items
}

func (c *PDBChecker) getPdbsAllowed() []int32 {
	pdbsAllowed := make([]int32, len(c.pdbItems))
	for i, pdbItem := range c.pdbItems {
		pdb := pdbItem.GetPDB()
		pdbsAllowed[i] = pdb.Status.DisruptionsAllowed
	}
	return pdbsAllowed
}

func (c *PDBChecker) check(_ internal.Graph, _ internal.EdgeSet, es ...internal.Edge) error {
	// TODO:
	// - Improve the performance.
	// - Evict pod (?)
	pdbsAllowed := c.getPdbsAllowed()
	for _, e := range es {
		pod := c.pods[e.GetName()]

		for i, pdbItem := range c.pdbItems {
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
				return fmt.Errorf("pod %s violating pdb %s(%d, %d)", podutil.GeneratePodKey(pod), util.GetPDBKey(pdb), pdb.Status.DisruptionsAllowed, pdbsAllowed[i]+1)
			}
		}
	}

	for i, pdbItem := range c.pdbItems {
		pdb := pdbItem.GetPDB()
		pdb.Status.DisruptionsAllowed = pdbsAllowed[i]
	}

	return nil
}

func (c *PDBChecker) GetCheckFunc() internal.CheckFunc {
	return c.check
}
