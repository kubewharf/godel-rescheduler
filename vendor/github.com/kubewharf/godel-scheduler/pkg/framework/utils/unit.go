/*
Copyright 2023 The Godel Scheduler Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"fmt"

	"github.com/kubewharf/godel-scheduler-api/pkg/client/listers/scheduling/v1alpha1"
	v1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/client-go/listers/scheduling/v1"
	"k8s.io/klog/v2"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	unitutil "github.com/kubewharf/godel-scheduler/pkg/util/unit"
)

// GetUnitIdentifier return id of the unit.
func GetUnitIdentifier(pod *v1.Pod) string {
	pgName := unitutil.GetPodGroupName(pod)
	if len(pgName) != 0 {
		return string(framework.PodGroupUnitType) + "/" + pod.Namespace + "/" + pgName
	}
	// pod doesn't belong to any unit. caller should handle this case
	key := string(framework.SinglePodUnitType) + "/" + pod.Namespace + "/" + pod.Name
	return key
}

// PodBelongToUnit check whether or not pod belongs to unit.
func PodBelongToUnit(pod *v1.Pod) bool {
	// TODO(jiaxin.shan@): retrieve registered units from runtime and use typed struct instead
	if _, exist := pod.Annotations[podutil.PodGroupNameAnnotationKey]; exist {
		return true
		// Support other units later
	} else {
		return false
	}
}

// GetUnitType return unit type of the pod. This method assumes pod belongs to unit
func GetUnitType(pod *v1.Pod) framework.ScheduleUnitType {
	if _, exist := pod.Annotations[podutil.PodGroupNameAnnotationKey]; exist {
		return framework.PodGroupUnitType
	}
	// TODO: Support other units later
	return framework.SinglePodUnitType
}

// CreateScheduleUnit create a unit object from the pod.
func CreateScheduleUnit(pcLister schedulingv1.PriorityClassLister, pgLister v1alpha1.PodGroupLister, info *framework.QueuedPodInfo) (framework.ScheduleUnit, error) {
	if len(unitutil.GetPodGroupName(info.Pod)) != 0 {
		pgName := unitutil.GetPodGroupName(info.Pod)
		podGroup, err := pgLister.PodGroups(info.Pod.Namespace).Get(pgName)
		if err != nil {
			return nil, fmt.Errorf("can not find pod group %s/%s, error: %v", info.Pod.Namespace, pgName, err)
		}

		var priority int32
		if len(podGroup.Spec.PriorityClassName) == 0 {
			priority = podutil.GetDefaultPriorityForGodelPod(info.Pod)
		} else {
			sc, err := pcLister.Get(podGroup.Spec.PriorityClassName)
			if err != nil {
				klog.ErrorS(err, "Failed to list PriorityClass")
				return nil, err
			}
			priority = sc.Value
		}

		// Can we introduce lister here to get priority and pass init Unit?
		unit := framework.NewPodGroupUnit(podGroup, priority)
		return unit, nil
	}

	return &framework.SinglePodUnit{}, nil
}
