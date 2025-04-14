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
	"sync"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	reschedulercache "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/runtime"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/runtime/registry"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/metrics"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/movementrecycler"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policyconfig"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/policycontroller"

	godelclient "github.com/kubewharf/godel-scheduler-api/pkg/client/clientset/versioned"
	crdinformers "github.com/kubewharf/godel-scheduler-api/pkg/client/informers/externalversions"
	schedulerlister "github.com/kubewharf/godel-scheduler-api/pkg/client/listers/scheduling/v1alpha1"
	commoncache "github.com/kubewharf/godel-scheduler/pkg/common/cache"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	pdbstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/pdb_store"
	preemptionstore "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/commonstores/preemption_store"
	schedulerframework "github.com/kubewharf/godel-scheduler/pkg/scheduler/framework"
	katalystinformers "github.com/kubewharf/katalyst-api/pkg/client/informers/externalversions"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
)

type Rescheduler struct {
	// It is expected that changes made via SchedulerCache will be observed
	// by NodeLister and Algorithm.
	cache *cache.ReschedulerCache

	informerFactory informers.SharedInformerFactory

	crdInformerFactory crdinformers.SharedInformerFactory

	pvcLister corelisters.PersistentVolumeClaimLister

	podLister corelisters.PodLister

	schedulerLister schedulerlister.SchedulerLister

	movementLister schedulerlister.MovementLister

	// Close this to shut down the scheduler.
	StopEverything <-chan struct{}

	// client syncs K8S object
	client clientset.Interface

	// crdClient syncs custom resources
	crdClient godelclient.Interface

	schedulerRegistry schedulerframework.Registry

	defaultSubClusterConfig *subClusterConfig
	subClusterConfigs       map[string]*subClusterConfig

	recorder events.EventRecorder

	//policyConfigs []*policyconfig.PolicyConfig

	movementRecycler *movementrecycler.MovementRecycler

	dryRun bool

	reschedulerPolicies []config.ReschedulerPolicy
	reschedulerRegistry registry.Registry

	metricsRecorder *runtime.MetricsRecorder
}

func (rs *Rescheduler) Run(ctx context.Context) {
	wait.UntilWithContext(ctx, rs.run, time.Second*10)
}

func (rs *Rescheduler) run(ctx context.Context) {
	// remove existing movements at first
	if err := rs.movementRecycler.CleanAllMovements(); err != nil {
		klog.Error(err)
		return
	}

	detectorPlugins := map[string]framework.Plugin{}
	algorithmPlugins := map[string]framework.Plugin{}
	var policyControllers []*policycontroller.PolicyController
	stopCh := ctx.Done()
	reschedLock := &sync.Mutex{}
	var generation uint = 0
	for _, config := range rs.reschedulerPolicies {
		klog.Infof("[Rescheduler] Start policy for scope: %v, detector: %s, algorithm: %v", config.Scope, config.Detector.Name, config.Algorithm.Name)

		movementTrigger := make(chan map[string]api.PodMovement)
		snapshotLister := reschedulercache.NewReschedulingSnapshotLister(5, rs.informerFactory, rs.crdInformerFactory)
		reschedulerForPolicy := NewReschedulerForPolicy(rs, nil, nil, nil, config.Scope)
		policyConfig := policyconfig.NewReschedulerPolicy(config, rs.reschedulerRegistry, reschedulerForPolicy)
		detectorPlugins[policyConfig.Detector.DetectorPlugin.Name()] = policyConfig.Detector.DetectorPlugin
		algorithmPlugins[policyConfig.Algorithm.Name()] = policyConfig.Algorithm
		policyController := policycontroller.New(rs.cache, movementTrigger, reschedLock, policyConfig, snapshotLister, rs.metricsRecorder, rs.recorder, reschedulerForPolicy, &generation, rs.dryRun, rs.crdClient, rs.client, rs.schedulerLister, rs.movementLister, rs.podLister)
		policyControllers = append(policyControllers, policyController)
	}

	for i, policyController := range policyControllers {
		if err := policyController.SetCheckPolicies(detectorPlugins, rs.reschedulerPolicies[i].Detector.CheckPolicies, cache.DetectorStore); err != nil {
			klog.Fatal(err)
		}
		if err := policyController.SetCheckPolicies(algorithmPlugins, rs.reschedulerPolicies[i].Algorithm.CheckPolicies, cache.AlgorithmStore); err != nil {
			klog.Fatal(err)
		}
		go policyController.Run(stopCh)
	}

	<-stopCh
}

// New returns a Scheduler
func New(
	client clientset.Interface,
	crdClient godelclient.Interface,
	informerFactory informers.SharedInformerFactory,
	crdInformerFactory crdinformers.SharedInformerFactory,
	katalystInformerFactory katalystinformers.SharedInformerFactory,
	stopCh <-chan struct{},
	recorder events.EventRecorder,
	schedulerConfig schedulerconfig.GodelSchedulerConfiguration,
	opts ...Option) (*Rescheduler, error,
) {
	metrics.Register()

	stopEverything := stopCh
	if stopEverything == nil {
		stopEverything = wait.NeverStop
	}

	options := renderOptions(opts...)
	podInformer := informerFactory.Core().V1().Pods()

	handlerWrapper := commoncache.MakeCacheHandlerWrapper().
		ComponentName(schedulerconfig.DefaultGodelSchedulerName).SchedulerType(schedulerconfig.DefaultSchedulerName).SubCluster(framework.DefaultSubCluster).
		PodAssumedTTL(15*time.Minute).Period(10*time.Second).StopCh(stopEverything).PodInformer(podInformer).PodLister(podInformer.Lister()).
		EnableStore(string(pdbstore.Name), string(preemptionstore.Name))
	reschedulerCache := cache.New(handlerWrapper.Obj())

	// Scheduler registry and plugins
	schedulerOpts := renderSchedulerOptions(
		WithDefaultProfile(schedulerConfig.DefaultProfile),
		WithSubClusterProfiles(schedulerConfig.SubClusterProfiles),
	)
	schedulerRegistry := schedulerframework.NewInTreeRegistry()
	defaultSubClusterConfig := newDefaultSubClusterConfig(schedulerOpts.defaultProfile)
	addSameTopologyFilterPlugin(defaultSubClusterConfig, schedulerRegistry)
	subClusterConfigs := map[string]*subClusterConfig{}
	for subClusterKey, subClusterProfile := range schedulerOpts.subClusterProfiles {
		subClusterConfig := newSubClusterConfigFromDefaultConfig(subClusterProfile, defaultSubClusterConfig)
		addSameTopologyFilterPlugin(subClusterConfig, schedulerRegistry)
		subClusterConfigs[subClusterKey] = subClusterConfig
	}
	rescheduler := &Rescheduler{
		cache:                   reschedulerCache,
		client:                  client,
		crdClient:               crdClient,
		informerFactory:         informerFactory,
		crdInformerFactory:      crdInformerFactory,
		pvcLister:               informerFactory.Core().V1().PersistentVolumeClaims().Lister(),
		podLister:               informerFactory.Core().V1().Pods().Lister(),
		schedulerLister:         crdInformerFactory.Scheduling().V1alpha1().Schedulers().Lister(),
		movementLister:          crdInformerFactory.Scheduling().V1alpha1().Movements().Lister(),
		StopEverything:          stopEverything,
		recorder:                recorder,
		dryRun:                  options.dryRun,
		reschedulerPolicies:     options.policyConfigs,
		reschedulerRegistry:     registry.NewInTreeRegistry(),
		schedulerRegistry:       schedulerRegistry,
		defaultSubClusterConfig: defaultSubClusterConfig,
		subClusterConfigs:       subClusterConfigs,
		metricsRecorder:         runtime.NewMetricsRecorder(1000, time.Second),
	}

	rescheduler.movementRecycler = movementrecycler.NewMovementRecycler(rescheduler.crdClient, rescheduler.movementLister)

	informerFactory.Core().V1().PersistentVolumes().Informer()
	informerFactory.Storage().V1().CSINodes().Informer()
	addAllEventHandlers(rescheduler, informerFactory, crdInformerFactory, katalystInformerFactory)
	return rescheduler, nil
}

func (r *Rescheduler) HasPlugin(pluginName string) bool {
	for _, plugin := range r.reschedulerPolicies {
		if plugin.Detector.Name == pluginName {
			return true
		} else if plugin.Algorithm.Name == pluginName {
			return true
		}
	}
	return false
}
