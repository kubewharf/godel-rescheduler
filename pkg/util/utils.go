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

package util

import (
	reschedulerconfig "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"

	"github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
)

func GetPDBKey(pdb *policy.PodDisruptionBudget) string {
	return pdb.Namespace + "/" + pdb.Name
}

func GetSubClusterForPod(pod *v1.Pod) string {
	return pod.Spec.NodeSelector[config.DefaultSubClusterKey]
}

func GetThrottleItemForInstance(instance *v1.Pod, throttleItems []reschedulerconfig.ThrottleItem) *reschedulerconfig.ThrottleItem {
	canBePreempted := podutil.CanPodBePreempted(instance) > 0
	var (
		defaultThrottle reschedulerconfig.ThrottleItem
		found           bool
	)
	for _, throttleItem := range throttleItems {
		if throttleItem.CanBePreempted != nil {
			if *throttleItem.CanBePreempted == canBePreempted {
				return &throttleItem
			}
		} else {
			defaultThrottle = throttleItem
			found = true
		}
	}
	if found {
		return &defaultThrottle
	}
	return nil
}
