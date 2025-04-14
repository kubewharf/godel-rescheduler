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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	cacheapi "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/api"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/runtime"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/metrics"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policyconfig"
	"github.com/kubewharf/godel-rescheduler/pkg/util/helper"
	podutil "github.com/kubewharf/godel-rescheduler/pkg/util/pod"

	"github.com/jasonlvhit/gocron"
	godelclient "github.com/kubewharf/godel-scheduler-api/pkg/client/clientset/versioned"
	schedulerlister "github.com/kubewharf/godel-scheduler-api/pkg/client/listers/scheduling/v1alpha1"
	schedulerframework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	godelpodutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog"
)

type PolicyController struct {
	policyConfig     *policyconfig.PolicyConfig
	reschedulerCache *cache.ReschedulerCache
	reschedLock      *sync.Mutex
	snapshotLister   *cache.ReschedulingSnapshotLister
	metricsRecorder  *runtime.MetricsRecorder
	eventRecorder    events.EventRecorder
	handle           api.ReschedulerFrameworkHandle
	generation       *uint
	dryRun           bool
	crdClient        godelclient.Interface
	client           clientset.Interface
	schedulerLister  schedulerlister.SchedulerLister
	movementLister   schedulerlister.MovementLister
	podLister        corelisters.PodLister
}

func New(
	reschedulerCache *cache.ReschedulerCache,
	movementTrigger chan map[string]api.PodMovement,
	reschedLock *sync.Mutex,
	policyConfig *policyconfig.PolicyConfig,
	snapshotLister *cache.ReschedulingSnapshotLister,
	metricsRecorder *runtime.MetricsRecorder,
	eventRecorder events.EventRecorder,
	handle api.ReschedulerFrameworkHandle,
	generation *uint,
	dryRun bool,
	crdClient godelclient.Interface,
	client clientset.Interface,
	schedulerLister schedulerlister.SchedulerLister,
	movementLister schedulerlister.MovementLister,
	podLister corelisters.PodLister,
) *PolicyController {
	return &PolicyController{
		reschedulerCache: reschedulerCache,
		policyConfig:     policyConfig,
		reschedLock:      reschedLock,
		snapshotLister:   snapshotLister,
		metricsRecorder:  metricsRecorder,
		eventRecorder:    eventRecorder,
		handle:           handle,
		generation:       generation,
		dryRun:           dryRun,
		crdClient:        crdClient,
		client:           client,
		schedulerLister:  schedulerLister,
		movementLister:   movementLister,
		podLister:        podLister,
	}
}

func (pc *PolicyController) Run(stopCh <-chan struct{}) {
	var allMovements *[]string = &[]string{}
	pc.run(stopCh, allMovements)
}

func (pc *PolicyController) run(stopCh <-chan struct{}, allMovements *[]string) {
	config := pc.policyConfig
	policyTrigger := config.PolicyTrigger
	if policyTrigger == nil {
		policyTrigger = config.Detector.DetectorTrigger
	}

	if policyTrigger == nil {
		return
	}

	task := func() {
		pc.reschedLock.Lock()
		defer func() {
			pc.reschedLock.Unlock()
		}()

		// UpdateSnapshot
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		pc.syncSnapshot(ctx)

		// run detector plugin to find all pods which need to be rescheduled (a global optimal algorithm returns nodes without pods)
		klog.Infof("[Rescheduler] Start detect for scope: %v, detector: %s, algorithm: %v", config.Scope, config.Detector.DetectorPlugin.Name(), config.Algorithm.Name())
		detectRes, err := pc.detect(ctx)
		if err != nil {
			klog.Infof("[Rescheduler] Detect failed for scope: %v, detector: %s, algorithm: %v, err: %v", config.Scope, config.Detector.DetectorPlugin.Name(), config.Algorithm.Name(), err)
			return
		}
		klog.Infof("[Rescheduler] Detect success for scope: %v, detector: %s, algorithm: %v, got %d nodes, detail: %#v", config.Scope, config.Detector.DetectorPlugin.Name(), config.Algorithm.Name(), len(detectRes.Nodes), detectRes)
		if len(detectRes.Nodes) == 0 {
			return
		}

		// reschedule
		klog.Infof("[Rescheduler] Start reschedule for scope: %v, detector: %s, algorithm: %v", config.Scope, config.Detector.DetectorPlugin.Name(), config.Algorithm.Name())
		podMovementDetail, err := pc.reschedule(ctx, *detectRes)
		if err != nil {
			klog.Infof("[Rescheduler] Reschedule failed for for scope: %v, detector: %s, algorithm: %v, err: %v", config.Scope, config.Detector.DetectorPlugin.Name(), config.Algorithm.Name(), err)
			return
		}
		klog.Infof("[Rescheduler] Reschedule success for scope: %v, detector: %s, algorithm: %v, momevement detail: %#v", config.Scope, config.Detector.DetectorPlugin.Name(), config.Algorithm.Name(), podMovementDetail)
		if len(podMovementDetail) == 0 {
			return
		}

		err = pc.checkMovementItems(ctx, podMovementDetail)
		if err != nil {
			klog.Errorf("[Rescheduler] Failed to check movement items: %v", err)
			return
		}

		err = pc.assumeInCache(ctx, podMovementDetail)
		if err != nil {
			klog.Errorf("[Rescheduler] Failed to assume in cache: %v", err)
			return
		}

		pc.makeMovements(podMovementDetail, allMovements)
	}

	// triggered after period
	if policyTrigger.Period != nil {
		go func() {
			time.Sleep(*policyTrigger.Period)
			wait.Until(task, *policyTrigger.Period, stopCh)
		}()
	}

	// triggered by cronjobs
	if len(policyTrigger.CronjobTriggers) > 0 {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					panic(err)
				}
			}()
			for _, crobjobTrigger := range policyTrigger.CronjobTriggers {
				gocron.Every(1).Day().At(crobjobTrigger.Time).Do(task)
			}
			stopCronCh := gocron.Start()
			<-stopCh
			close(stopCronCh)
		}()
	}

	// triggered by signal
	if policyTrigger.Signal != nil {
		signalTrigger := make(chan os.Signal, 1)
		signal.Notify(signalTrigger, syscall.Signal(*policyTrigger.Signal))
		go func() {
			defer func() {
				if err := recover(); err != nil {
					panic(err)
				}
			}()
			for {
				select {
				case <-stopCh:
					return
				case <-signalTrigger:
					task()
				}
			}
		}()
	}

	// triggered by http call
	if policyTrigger.Port != nil {
		go func() {
			addr := fmt.Sprintf(":%d", *policyTrigger.Port)
			indexHandler := func(w http.ResponseWriter, r *http.Request) {
				task()
			}
			handler := http.NewServeMux()
			handler.HandleFunc("/trigger", indexHandler)
			srv := &http.Server{Addr: addr, Handler: handler}
			go func() {
				defer func() {
					if err := recover(); err != nil {
						panic(err)
					}
				}()
				if err := srv.ListenAndServe(); err != nil {
					klog.Warningf("failed to listen and serve addr %s: %v", addr, err)
				}
			}()
			<-stopCh
			srv.Shutdown(context.Background())
		}()
	}

	<-stopCh
}

func (pc *PolicyController) detect(ctx context.Context) (*api.DetectorResult, error) {
	plugin := pc.policyConfig.Detector.DetectorPlugin

	startTime := time.Now()
	pc.setSnapshot(ctx, cache.DetectorStore)
	detectRes, err := plugin.Detect(ctx)
	pc.metricsRecorder.ObserveOperationDurationAsync(metrics.DetectorEvaluation, plugin.Name(), helper.SinceInSeconds(startTime))
	if err != nil {
		return nil, fmt.Errorf("failed to run detect plugin: %s: %v", plugin.Name(), err)
	}
	return &detectRes, nil
}

func (pc *PolicyController) reschedule(ctx context.Context, detectRes api.DetectorResult) (map[string]api.PodMovement, error) {
	plugin := pc.policyConfig.Algorithm
	// use changed nodes
	startTime := time.Now()
	pc.setSnapshot(ctx, cache.AlgorithmStore)
	res, err := plugin.Reschedule(ctx, detectRes)
	pc.metricsRecorder.ObserveOperationDurationAsync(metrics.AlgorithmEvaluation, plugin.Name(), helper.SinceInSeconds(startTime))
	if err != nil {
		return nil, fmt.Errorf("failed to run algorithm plugin: %s: %v", plugin.Name(), err)
	}
	if len(res.PodMovements) == 0 {
		return nil, fmt.Errorf("nil reschedule result")
	}

	pc.setSnapshot(ctx, cache.AlgorithmStore)
	movementsDetails := map[string]api.PodMovement{}
	for _, podMovement := range res.PodMovements {
		podKey := godelpodutil.GeneratePodKey(podMovement.Pod)
		movementsDetails[podKey] = podMovement
		if err := pc.handle.RemovePod(podMovement.Pod, cache.AlgorithmStore); err != nil {
			return nil, fmt.Errorf("failed to remove pod %s from node %s: %v", podKey, podMovement.PreviousNode, err)
		}
	}

	failureCountMap := map[string]int{}
	failureCount := 0
	for _, podMovement := range res.PodMovements {
		podKey := godelpodutil.GeneratePodKey(podMovement.Pod)
		podCopy := podutil.PrepareReschedulePod(podMovement.Pod)
		// check result accuracy
		if podMovement.TargetNode == "" {
			continue
		}
		nodeInfo, err := pc.handle.GetNodeInfo(podMovement.TargetNode)
		if err != nil {
			return nil, err
		}
		fitNodes, statusMap, err := pc.handle.FindNodesThatFitPod(ctx, podCopy, []schedulerframework.NodeInfo{nodeInfo})
		if err != nil {
			msg := err.Error()
			klog.Errorf("[Rescheduler] Failed to move pod %s from %s to %s: %s", podKey, podMovement.PreviousNode, podMovement.TargetNode, msg)
			failureCountMap[msg]++
			failureCount++
		} else if len(fitNodes) == 0 {
			msg := statusMap[podMovement.TargetNode].Message()
			klog.Errorf("[Rescheduler] Failed to move pod %s from %s to %s: %s", podKey, podMovement.PreviousNode, podMovement.TargetNode, msg)
			failureCountMap[formatMetricsTagValue(msg)]++
			failureCount++
		} else {
			if err := pc.handle.AssumePod(podCopy, podMovement.TargetNode, cache.AlgorithmStore); err != nil {
				return nil, fmt.Errorf("failed to assume pod %s to node %s: %v", podKey, podMovement.TargetNode, err)
			}
		}
	}
	metrics.ObserveRescheduledPodCountInAllMovements(plugin.Name(), "all", "nil", len(res.PodMovements))
	for reason, count := range failureCountMap {
		metrics.ObserveRescheduledPodCountInAllMovements(plugin.Name(), "fail", reason, count)
	}
	failureRatio := float64(failureCount) / float64(len(res.PodMovements))
	config := pc.policyConfig
	if failureRatio > config.FailureRatioThreshold {
		return nil, fmt.Errorf("got failure ratio %d/%d which is larger than threshold %f", failureCount, len(res.PodMovements), config.FailureRatioThreshold)
	}
	klog.Infof("[Rescheduler] Reschedule success for scope: %v, detector: %s, algorithm: %v, failure ratio: %d/%d", config.Scope, config.Detector.DetectorPlugin.Name(), plugin.Name(), failureCount, len(res.PodMovements))
	return movementsDetails, nil
}

func formatMetricsTagValue(msg string) string {
	r := strings.NewReplacer(" ", "_", ",", "_", ";", "_", ":", "_", "(", "/", ")", "/", "'", "/", "{", "/", "}", "/", "[", "/", "]", "/")
	msg = r.Replace(msg)
	if len(msg) < 1 {
		msg = "nil"
	} else if len(msg) > 255 {
		msg = msg[0:255]
	}
	return msg
}

func (pc *PolicyController) SetCheckPolicies(policies map[string]schedulerframework.Plugin, checkPolicies []string, storeType string) error {
	var policyStoreSyncerPlugins []api.PolicyStoreSyncerPlugin
	for _, checkPolicy := range checkPolicies {
		if _, ok := policies[checkPolicy]; !ok {
			return fmt.Errorf("check policy %s not exist", checkPolicy)
		}
		policyStoreSyncerPlugins = append(policyStoreSyncerPlugins, policies[checkPolicy].(api.PolicyStoreSyncerPlugin))
	}
	switch storeType {
	case cache.DetectorStore:
		pc.policyConfig.CheckPoliciesForDetector = policyStoreSyncerPlugins
	case cache.AlgorithmStore:
		pc.policyConfig.CheckPoliciesForAlgorithm = policyStoreSyncerPlugins
	}
	return nil
}

func (pc *PolicyController) syncSnapshot(ctx context.Context) {
	pc.reschedulerCache.UpdateSnapshots(pc.snapshotLister)
}

func (pc *PolicyController) setSnapshot(ctx context.Context, storeType string) {
	snapshot := pc.snapshotLister.Pop()
	var plugins []api.PolicyStoreSyncerPlugin
	switch storeType {
	case cache.DetectorStore:
		plugins = pc.policyConfig.CheckPoliciesForDetector
	case cache.AlgorithmStore:
		plugins = pc.policyConfig.CheckPoliciesForAlgorithm
	}
	for _, checkPolicy := range plugins {
		store := checkPolicy.SyncPolicyStore(ctx, snapshot)
		snapshot.SyncPolicyStore(checkPolicy.Name(), store, storeType)
	}
	pc.handle.SetSnapshot(snapshot)
	pc.handle.GeneratePluginRegistryMap()
}

func (pc *PolicyController) checkMovementItems(ctx context.Context, movementsDetails map[string]api.PodMovement) error {
	pc.setSnapshot(ctx, cache.AlgorithmStore)
	for _, item := range movementsDetails {
		if err := pc.handle.RemovePod(item.Pod, cache.AlgorithmStore); err != nil {
			return fmt.Errorf("failed to remove pod %s: %v", godelpodutil.GeneratePodKey(item.Pod), err)
		}
	}
	for _, item := range movementsDetails {
		if policyName, res := pc.handle.CheckMovementItem(item.PreviousNode, item.TargetNode, item.Pod, cache.AlgorithmStore); res != policies.Pass {
			return fmt.Errorf("check movement item(pod %s form %s to %s) failed in policy %s: %v", godelpodutil.GeneratePodKey(item.Pod), item.PreviousNode, item.TargetNode, policyName, res)
		}
		podCopy := podutil.PrepareReschedulePod(item.Pod)
		pc.handle.AssumePod(podCopy, item.TargetNode, cache.AlgorithmStore)
	}
	return nil
}

func (pc *PolicyController) assumeInCache(ctx context.Context, movementsDetails map[string]api.PodMovement) error {
	for _, item := range movementsDetails {
		podCopy := podutil.PrepareReschedulePod(item.Pod)
		pInfo := schedulerframework.MakeCachePodInfoWrapper().Pod(podCopy).Obj()
		reschedPodInfo := cacheapi.MakeCachePodInfoWrapper().PodInfo(pInfo).MovementGeneration(*pc.generation).Obj()
		if err := pc.reschedulerCache.AssumePod(reschedPodInfo); err != nil {
			return err
		}
	}
	return nil
}
