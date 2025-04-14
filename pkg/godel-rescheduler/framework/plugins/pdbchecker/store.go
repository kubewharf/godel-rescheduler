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

package pdbchecker

import (
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/util"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/plugins/preempting/pdbchecker"
	v1 "k8s.io/api/core/v1"
)

type pdbCheckerStore struct {
	cache *pdbCheckerCache
}

var _ policies.PolicyStore = &pdbCheckerStore{}

func newPDBCheckerStore(snapshot *cache.ReschedulingSnapshot) policies.PolicyStore {
	return &pdbCheckerStore{
		cache: generateCache(snapshot),
	}
}

func (pdb *pdbCheckerStore) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod) policies.CheckResult {
	match, violating, _ := checkPDB(instance, pdb.cache)
	if match {
		if violating {
			return policies.MovedInstanceError
		}
		return policies.Pass
	}

	violating, _ = pdbchecker.CheckPodDisruptionBudgetViolation(instance, pdb.cache.disruptionsAllowed, pdb.cache.pdbItems)
	if violating {
		return policies.MovedInstanceError
	}
	return policies.Pass
}

func (pdb *pdbCheckerStore) AssumePod(pod *v1.Pod) {
	match, _, index := checkPDB(pod, pdb.cache)
	if match {
		pdb.cache.disruptionsAllowed[index]--
		return
	}

	_, indexes := pdbchecker.CheckPodDisruptionBudgetViolation(pod, pdb.cache.disruptionsAllowed, pdb.cache.pdbItems)
	for _, index := range indexes {
		pdb.cache.disruptionsAllowed[index]--
	}
}

func (pdb *pdbCheckerStore) RemovePod(pod *v1.Pod) {}

type pdbCheckerCache struct {
	pdbItems           []framework.PDBItem
	disruptionsAllowed []int32
	indexMap           map[string]int
}

func generateCache(snapshot *cache.ReschedulingSnapshot) *pdbCheckerCache {
	pdbItems := snapshot.GetPDBItemList()
	disruptionsAllowed := make([]int32, len(pdbItems))
	indexMap := map[string]int{}
	for i, pdb := range pdbItems {
		disruptionsAllowed[i] = pdb.GetPDB().Status.DisruptionsAllowed
		pdbKey := util.GetPDBKey(pdb.GetPDB())
		indexMap[pdbKey] = i
	}
	return &pdbCheckerCache{
		pdbItems:           pdbItems,
		disruptionsAllowed: disruptionsAllowed,
		indexMap:           indexMap,
	}
}

// only one matched pdb for a pod
// match, violate
func checkPDB(instance *v1.Pod, pdbCheckerCache *pdbCheckerCache) (bool, bool, int) {
	pdbKey := instance.Namespace + "/" + instance.Labels["name"]
	index, ok := pdbCheckerCache.indexMap[pdbKey]
	if !ok {
		return false, false, -1
	}
	pdbItem := pdbCheckerCache.pdbItems[index]
	if !pdbchecker.PDBMatched(instance, pdbItem) {
		return false, false, -1
	}
	return true, pdbCheckerCache.disruptionsAllowed[index] <= 0, index
}
