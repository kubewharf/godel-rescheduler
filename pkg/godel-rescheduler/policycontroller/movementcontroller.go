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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/metrics"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/checker"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller/generator/relation"
	"github.com/kubewharf/godel-rescheduler/pkg/util/helper"
	"github.com/kubewharf/godel-rescheduler/pkg/util/parallelize"
	podutil "github.com/kubewharf/godel-rescheduler/pkg/util/pod"

	schedulingv1alpha1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/scheduling/v1alpha1"
	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	"github.com/kubewharf/godel-scheduler/pkg/framework/utils"
	pdbstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/pdb_store"
	preemptionstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/preemption_store"
	godelpodutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const (
	EvictionSucceed  = "succeed"
	EvictionFail     = "fail"
	EvictionNotFound = "not_found"
	EvictionDelay    = "delay"
)

func (pc *PolicyController) makeMovements(podMovementDetails map[string]api.PodMovement, allMovements *[]string) {
	// clean movements generated in previous generation
	pc.cleanMovements(*allMovements)
	klog.Infof("[Rescheduler] start process movements for %s", pc.policyConfig.Algorithm.Name())
	startTime := time.Now()
	if err := pc.checkMovementsInAdvance(podMovementDetails); err != nil {
		klog.Warningf("[Rescheduler] could not generate movements in advance: %v", err)
		return
	}
	*allMovements = pc.generateAndProcessMovements(podMovementDetails, *pc.generation)
	*pc.generation++
	pc.metricsRecorder.ObserveOperationDurationAsync(metrics.MovementControllerEvaluation, pc.policyConfig.Algorithm.Name(), helper.SinceInSeconds(startTime))
	klog.Infof("[Rescheduler] process movements for %s end", pc.policyConfig.Algorithm.Name())
}

func (pc *PolicyController) checkMovementsInAdvance(podMovementDetails map[string]api.PodMovement) error {
	checker := checker.NewPDBChecker()
	snapshot := pc.snapshotLister.Pop()
	pc.reschedulerCache.UpdateSnapshot(snapshot)

	allRelations := generateRelations(podMovementDetails)
	generator := generator.NewGenerator(context.Background(), allRelations, checker.GetCheckFunc())
	for !generator.Finished() {
		checker.Refresh(getPods(podMovementDetails), snapshot.GetPDBItemList())
		if _, err := generator.Generate(context.Background()); err != nil {
			return err
		}
	}
	return nil
}

func (pc *PolicyController) generateAndProcessMovements(
	podMovementDetails map[string]api.PodMovement,
	generation uint,
) []string {
	var allMovements []string
	allRelations := generateRelations(podMovementDetails)
	checker := checker.NewPDBChecker()
	generator := generator.NewGenerator(context.Background(), allRelations, checker.GetCheckFunc())
	failedCount := 0

	handlerWrapper := commoncache.MakeCacheHandlerWrapper().EnableStore(string(pdbstore.Name), string(preemptionstore.Name))
	snapshot := cache.NewEmptySnapshot(handlerWrapper.Obj())

	for !generator.Finished() {
		klog.Infof("[Rescheduler] start process each movement for %s", pc.policyConfig.Algorithm.Name())
		pc.reschedulerCache.UpdateSnapshot(snapshot)
		// check if failure ratio is higer then threhold
		if failureRatio, err := pc.getMovementFailureRatio(allMovements); err != nil {
			klog.Errorf("[Rescheduler] failed to get failure ratio: %v", err)
			break
		} else if failureRatio > pc.policyConfig.FailureRatioThreshold {
			klog.Infof("[Rescheduler] failure ratio %f is larger then threshold %f, break this generation", failureRatio, pc.policyConfig.FailureRatioThreshold)
			break
		}

		checker.Refresh(getPods(podMovementDetails), snapshot.GetPDBItemList())
		relations, err := generator.Generate(context.Background())
		if err != nil {
			if failedCount > 10 {
				klog.Errorf("[Rescheduler] failed to generate movements: %v, and timeout", err)
				break
			}
			klog.Errorf("[Rescheduler] failed to generate movements: %v, retry...", err)
			failedCount++
			time.Sleep(3 * time.Second)
			continue
		}

		failedCount = 0
		movement := pc.generateMovement(relations, podMovementDetails, generation)
		allMovements = append(allMovements, movement.Name)

		if err := pc.processMovement(movement); err != nil {
			klog.Errorf("[Rescheduler] failed to process movement for %s: %v", pc.policyConfig.Algorithm.Name(), err)
			break
		}
		klog.Infof("[Rescheduler] process movement %s success for %s", movement.Name, pc.policyConfig.Algorithm.Name())
	}
	return allMovements
}

func generateRelations(podMovementDetails map[string]api.PodMovement) relation.Relations {
	var relations relation.Relations
	for _, podMovementDetail := range podMovementDetails {
		relation := relation.MakeRelation()
		relation.From(podMovementDetail.PreviousNode)
		relation.To(podMovementDetail.TargetNode)
		podKey := godelpodutil.GeneratePodKey(podMovementDetail.Pod)
		relation.Name(podKey)
		relations = append(relations, relation.Obj())
	}
	return relations
}

func getPods(podMovementDetails map[string]api.PodMovement) map[string]*corev1.Pod {
	pods := map[string]*corev1.Pod{}
	for podKey, podDetail := range podMovementDetails {
		pods[podKey] = podDetail.Pod
	}
	return pods
}

// generate movement crd from relations
func (pc *PolicyController) generateMovement(relations relation.Relations, podMovementDetails map[string]api.PodMovement, generation uint) *schedulingv1alpha1.Movement {
	movement := &schedulingv1alpha1.Movement{
		Spec: schedulingv1alpha1.MovementSpec{
			Creator:    pc.policyConfig.Algorithm.Name(),
			Generation: generation,
		},
	}
	var deletedTasks []*schedulingv1alpha1.TaskInfo
	ownerMovementDetails := map[string]*OwnerMovementDetail{}
	for _, relation := range relations {
		podKey := relation.GetName()
		podDetail := podMovementDetails[podKey]
		taskInfo := utils.GetTaskInfoFromPod(podDetail.Pod)
		deletedTasks = append(deletedTasks, taskInfo)

		owner := godelpodutil.GetPodOwnerInfo(podDetail.Pod)
		if owner == nil {
			continue
		}
		ownerKey := godelpodutil.GetOwnerInfoKey(owner)
		ownerMovementDetail := ownerMovementDetails[ownerKey]
		if ownerMovementDetail == nil {
			ownerMovementDetails[ownerKey] = &OwnerMovementDetail{
				OwnerInfo:         owner,
				NodeSuggestionMap: map[string]*schedulingv1alpha1.RecommendedNode{},
			}
			ownerMovementDetail = ownerMovementDetails[ownerKey]
		}
		targetNode := podDetail.TargetNode
		// if not set suggestion node, continue
		if targetNode == "" {
			continue
		}
		nodeSuggestion := ownerMovementDetail.NodeSuggestionMap[targetNode]
		if nodeSuggestion == nil {
			ownerMovementDetail.NodeSuggestionMap[targetNode] = &schedulingv1alpha1.RecommendedNode{
				Node: targetNode,
			}
			nodeSuggestion = ownerMovementDetail.NodeSuggestionMap[targetNode]
		}
		nodeSuggestion.DesiredPodCount++
	}
	movement.Spec.DeletedTasks = deletedTasks
	for _, ownerMovementDetail := range ownerMovementDetails {
		ownerMovement := &schedulingv1alpha1.Owner{
			Owner: ownerMovementDetail.OwnerInfo,
		}
		for _, nodeSuggestion := range ownerMovementDetail.NodeSuggestionMap {
			ownerMovement.RecommendedNodes = append(ownerMovement.RecommendedNodes, nodeSuggestion)
		}
		movement.Status.Owners = append(movement.Status.Owners, ownerMovement)
	}
	movement.Name = generateMovementName(movement)
	return movement
}

func (pc *PolicyController) cleanMovements(movements []string) {
	for _, movement := range movements {
		if deleteErr := pc.deleteMovement(movement); deleteErr != nil {
			klog.Errorf("[Rescheduler] Failed to delete movement %s: %v", movement, deleteErr)
		}
	}
}

func (pc *PolicyController) processMovement(movement *schedulingv1alpha1.Movement) error {
	// apply movement crd to apiserver
	if err := pc.createMovement(movement); err != nil {
		return fmt.Errorf("failed to create movement %s: %v", movement.Name, err)
	}
	// check movement status
	// check if movement has been catched by all godel scheduler instances
	if err := pc.checkMovementCatched(movement.Name); err != nil {
		return fmt.Errorf("failed to check movement %s: %v", movement.Name, err)
	}
	if pc.dryRun {
		return nil
	}
	// kill pods
	if err := pc.killTasksInEachMovement(movement); err != nil {
		return fmt.Errorf("failed to kill tasks in movement %s: %v", movement.Name, err)
	}

	return nil
}

func (pc *PolicyController) createMovement(movement *schedulingv1alpha1.Movement) error {
	if err := retry.OnError(DefaultRetry,
		func(err error) bool {
			return true
		},
		func() error {
			var (
				gotMovement       *schedulingv1alpha1.Movement
				gotErr, createErr error
			)
			if gotMovement, gotErr = pc.crdClient.SchedulingV1alpha1().Movements().Get(context.Background(), movement.Name, v1.GetOptions{}); gotErr != nil {
				if !errors.IsNotFound(gotErr) {
					return gotErr
				}
				// create movement
				if gotMovement, createErr = pc.crdClient.SchedulingV1alpha1().Movements().Create(context.Background(), movement, v1.CreateOptions{}); createErr != nil {
					return createErr
				}
			}

			oldData, err := json.Marshal(gotMovement)
			if err != nil {
				return err
			}
			newMovement := gotMovement.DeepCopy()
			newMovement.Status = movement.Status
			newData, err := json.Marshal(newMovement)
			if err != nil {
				return err
			}
			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, &schedulingv1alpha1.Movement{})
			if err != nil {
				return err
			}

			_, err = pc.crdClient.SchedulingV1alpha1().Movements().Patch(context.Background(), movement.Name, types.MergePatchType, patchBytes, v1.PatchOptions{}, "status")
			return err
		}); err != nil {
		return err
	}
	metrics.ObserveRescheduledPodCountInEachMovement(pc.policyConfig.Algorithm.Name(), "all", len(movement.Spec.DeletedTasks))
	return nil
}

func (pc *PolicyController) checkMovementCatched(name string) error {
	return retry.OnError(RetryMultipleTimes,
		func(err error) bool {
			return true
		},
		func() error {
			movement, getErr := pc.movementLister.Get(name)
			if getErr != nil {
				return getErr
			}

			notifiedSchedulers := sets.NewString(movement.Status.NotifiedSchedulers...)
			allSchedulers, err := pc.schedulerLister.List(labels.Everything())
			if err != nil {
				return err
			}
			if len(allSchedulers) == 0 {
				return fmt.Errorf("no godel scheduler in cluster")
			}
			for _, scheduler := range allSchedulers {
				if !notifiedSchedulers.Has(scheduler.Name) {
					return fmt.Errorf("Notified schedulers: %v, missing scheduler %v", notifiedSchedulers, scheduler.Name)
				}
			}
			return nil
		})
}

func (pc *PolicyController) killTasksInEachMovement(movement *schedulingv1alpha1.Movement) error {
	var (
		evictionResult = newEvictionResult()
		evictionErrs   []error
	)
	defer func() {
		evictionResult.RecordMetrics(pc.policyConfig.Algorithm.Name(), movement.Name)
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	evictPod := func(i int) {
		deletedTask := movement.Spec.DeletedTasks[i]
		podUID := deletedTask.UID
		gotPod, gotErr := pc.podLister.Pods(deletedTask.Namespace).Get(deletedTask.Name)
		if gotErr != nil {
			if errors.IsNotFound(gotErr) {
				return
			}
			evictionResult.AddResult(EvictionFail)
			evictionErrs = append(evictionErrs, gotErr)
			cancel()
			return
		}
		if gotPod.UID != podUID {
			return
		}
		// if timeout, return directly
		stopTimer := make(chan struct{})
		go func() {
			// use evict function in order to not violating pdb
			if evictErr := podutil.EvictPod(gotPod, pc.client, pc.eventRecorder, pc.policyConfig.Algorithm.Name()); evictErr != nil {
				if errors.IsNotFound(evictErr) {
					evictionResult.AddResult(EvictionNotFound)
					klog.Infof("ignore eviction error for pod %s: %v", godelpodutil.GeneratePodKey(gotPod), evictErr)
				} else if IsDelayEviction(evictErr) {
					evictionResult.AddResult(EvictionDelay)
					klog.Infof("ignore eviction error for pod %s: %v", godelpodutil.GeneratePodKey(gotPod), evictErr)
				} else {
					evictionResult.AddResult(EvictionFail)
					evictionErrs = append(evictionErrs, fmt.Errorf("Failed to evict pod %s/%s/%s in movement %s: %v", deletedTask.Namespace, deletedTask.Name, deletedTask.UID, movement.Name, evictErr))
					cancel()
				}
			} else {
				evictionResult.AddResult(EvictionSucceed)
			}
			close(stopTimer)
		}()
		timer := time.NewTimer(time.Minute)
		defer timer.Stop()
		select {
		case <-timer.C:
			evictionResult.AddResult(EvictionFail)
			evictionErrs = append(evictionErrs, fmt.Errorf("Failed to evict pod %s/%s/%s in movement %s: timeout", deletedTask.Namespace, deletedTask.Name, deletedTask.UID, movement.Name))
			cancel()
		case <-stopTimer:
		}
	}

	parallelize.Until(ctx, len(movement.Spec.DeletedTasks), evictPod)
	if ctx.Err() != nil {
		return utilerrors.NewAggregate(evictionErrs)
	}
	return nil
}

func (pc *PolicyController) getMovementFailureRatio(movements []string) (float64, error) {
	var allTasks, failureTasks int
	for _, movement := range movements {
		if err := retry.OnError(RetryEvenMoreTimes,
			func(err error) bool {
				return true
			},
			func() error {
				movement, getErr := pc.movementLister.Get(movement)
				if getErr != nil {
					return getErr
				}
				for _, ownerMovement := range movement.Status.Owners {
					for _, nodeSuggestion := range ownerMovement.RecommendedNodes {
						allTasks += int(nodeSuggestion.DesiredPodCount)
						if len(nodeSuggestion.ActualPods) > int(nodeSuggestion.DesiredPodCount) {
							failureTasks += len(nodeSuggestion.ActualPods) - int(nodeSuggestion.DesiredPodCount)
						}
					}
					failureTasks += len(ownerMovement.MismatchedTasks)
				}
				return nil
			}); err != nil {
			return 0, err
		}
	}
	return float64(failureTasks) / float64(allTasks), nil
}

func (pc *PolicyController) deleteMovement(movementName string) error {
	return retry.OnError(DefaultRetry,
		func(err error) bool {
			return true
		},
		func() error {
			_, getErr := pc.movementLister.Get(movementName)
			if getErr != nil && errors.IsNotFound(getErr) {
				return nil
			}
			deleteErr := pc.crdClient.SchedulingV1alpha1().Movements().Delete(context.Background(), movementName, v1.DeleteOptions{})
			return deleteErr
		})
}

type evictionResult struct {
	result map[string]int
	mu     sync.RWMutex
}

func newEvictionResult() *evictionResult {
	return &evictionResult{
		result: map[string]int{},
	}
}

func (r *evictionResult) RecordMetrics(algorithmName, movementName string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for msg, count := range r.result {
		if count > 0 {
			metrics.ObserveRescheduledPodCountInEachMovement(algorithmName, msg, count)
		}
	}
}

func (r *evictionResult) AddResult(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.result[msg]++
}

func IsDelayEviction(err error) bool {
	return strings.Contains(err.Error(), "In process of NegotiatedPodEviction")
}
