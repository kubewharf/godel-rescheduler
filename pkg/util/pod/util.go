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

package pod

import (
	"context"
	"fmt"

	"github.com/kubewharf/godel-scheduler/pkg/framework/utils"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/events"
)

const (
	// PodNodeBeforeRescheduleAnnotationKey is a pod annotation key, value is the pod's node before reschedule.
	PodNodeBeforeRescheduleAnnotationKey = "godel.bytedance.com/node-before-reschedule"
)

func ConvertToPod(obj interface{}) (*v1.Pod, error) {
	var pod *v1.Pod
	switch t := obj.(type) {
	case *v1.Pod:
		pod = obj.(*v1.Pod)
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*v1.Pod)
		if !ok {
			return nil, fmt.Errorf("unable to convert object %T to *v1.Pod", obj)
		}
	default:
		return nil, fmt.Errorf("unable to handle object: %T", obj)
	}
	return pod, nil
}

func GetPodNodeBeforeReschedule(pod *v1.Pod) string {
	if pod == nil || pod.Annotations == nil {
		return ""
	}
	return pod.Annotations[PodNodeBeforeRescheduleAnnotationKey]
}

func EvictPod(pod *v1.Pod, client clientset.Interface, eventRecorder events.EventRecorder, policyName string) error {
	var (
		policyGroupVersion = "policy/v1beta1"
		evictionKind       = "Eviction"
	)

	deleteOptions := &metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{
			UID: &pod.UID,
		},
	}
	eviction := &policy.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyGroupVersion,
			Kind:       evictionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: deleteOptions,
	}
	msg := fmt.Sprintf("Pod is evicted by rescheduling policy %s", policyName)
	eventRecorder.Eventf(pod, nil, v1.EventTypeWarning, "Rescheduling", "Evicted", msg)
	// Remember to change change the URL manipulation func when Evction's version change
	return client.PolicyV1beta1().Evictions(eviction.Namespace).Evict(context.Background(), eviction)
}

func PrepareReschedulePod(pod *v1.Pod) *v1.Pod {
	podCopy := pod.DeepCopy()

	// 1. Set pod's node before reschedule.
	if podCopy.Annotations == nil {
		podCopy.Annotations = map[string]string{}
	}
	podCopy.Annotations[PodNodeBeforeRescheduleAnnotationKey] = utils.GetNodeNameFromPod(pod)

	// 2. Clean pod annotations and pod spec.
	delete(podCopy.Annotations, podutil.AssumedNodeAnnotationKey)
	delete(podCopy.Annotations, podutil.AssumedCrossNodeAnnotationKey)
	delete(podCopy.Annotations, podutil.NominatedNodeAnnotationKey)
	delete(podCopy.Annotations, podutil.FailedSchedulersAnnotationKey)
	delete(podCopy.Annotations, podutil.MicroTopologyKey)
	delete(podCopy.Annotations, podutil.SchedulerAnnotationKey)
	delete(podCopy.Annotations, podutil.MicroTopologyKey)
	podCopy.Annotations[podutil.PodStateAnnotationKey] = string(podutil.PodPending)
	podCopy.Spec.NodeName = ""
	for i := range podCopy.Spec.Containers {
		podCopy.Spec.Containers[i].Ports = nil
	}
	return podCopy
}
