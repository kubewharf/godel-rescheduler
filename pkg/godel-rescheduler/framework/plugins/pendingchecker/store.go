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

package pendingchecker

import (
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	reschedulerutil "github.com/kubewharf/godel-rescheduler/pkg/util"

	v1 "k8s.io/api/core/v1"
)

type pendingCheckerStore struct {
	pendingPods map[string]int
	args        *config.PendingCheckerArgs
}

var _ policies.PolicyStore = &pendingCheckerStore{}

func newPendingCheckerStore(pendingPods map[string]int, args *config.PendingCheckerArgs) policies.PolicyStore {
	return &pendingCheckerStore{
		pendingPods: pendingPods,
		args:        args,
	}
}

func (pcs *pendingCheckerStore) CheckMovementItem(previousNode, targetNode string, instance *v1.Pod) policies.CheckResult {
	if pcs.args == nil {
		return policies.Pass
	}

	subCluster := reschedulerutil.GetSubClusterForPod(instance)
	pendingPodCount := pcs.getPendingPodCount(subCluster)

	if threhold, ok := pcs.args.PendingThresholds[subCluster]; ok {
		return getResult(pendingPodCount, threhold)
	} else if pcs.args.DefaultPendingThreshold != nil {
		return getResult(pendingPodCount, *pcs.args.DefaultPendingThreshold)
	}
	return policies.Pass
}

func getResult(pendingPodCount, threshold int) policies.CheckResult {
	if pendingPodCount > threshold {
		return policies.MovedInstanceError
	}
	return policies.Pass
}

func (pcs *pendingCheckerStore) getPendingPodCount(subCluster string) int {
	return pcs.pendingPods[subCluster]
}

func (pcs *pendingCheckerStore) AssumePod(pod *v1.Pod) {}

func (pcs *pendingCheckerStore) RemovePod(pod *v1.Pod) {}
