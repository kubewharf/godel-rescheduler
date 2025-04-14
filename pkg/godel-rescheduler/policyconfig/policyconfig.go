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

package policyconfig

import (
	"time"

	reschedulerconfig "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	framework "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/runtime/registry"

	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
)

const (
	DefaultFailureRatioThreshold = 0.2
)

type PolicyConfig struct {
	PolicyTrigger             *api.PolicyTrigger
	Detector                  Detector
	Algorithm                 api.AlgorithmPlugin
	CheckPoliciesForDetector  []api.PolicyStoreSyncerPlugin
	CheckPoliciesForAlgorithm []api.PolicyStoreSyncerPlugin
	// Scope is node selector
	Scope                 map[string]string
	FailureRatioThreshold float64
}

type Detector struct {
	DetectorTrigger *api.PolicyTrigger // for backward compatibility
	DetectorPlugin  api.DetectorPlugin
}

func NewReschedulerPolicy(config reschedulerconfig.ReschedulerPolicy, registry registry.Registry, rfh framework.ReschedulerFrameworkHandle) *PolicyConfig {
	policy := &PolicyConfig{}
	policy.PolicyTrigger = convertPolicyTrigger(config.PolicyTrigger)
	policy.Scope = config.Scope
	policy.Detector = convertDetector(config.Detector, registry, rfh)
	policy.Algorithm = newAlgorithmPlugin(registry, config.Algorithm.Name, config.Algorithm.Args, rfh)
	policy.FailureRatioThreshold = DefaultFailureRatioThreshold
	if config.FailureRatioThreshold != nil {
		policy.FailureRatioThreshold = *config.FailureRatioThreshold
	}
	return policy
}

func convertDetector(detectorConfig reschedulerconfig.DetectorConfig, registry registry.Registry, rfh framework.ReschedulerFrameworkHandle) Detector {
	var detector Detector
	detector.DetectorTrigger = convertPolicyTrigger(detectorConfig.DetectorTrigger)
	detector.DetectorPlugin = newDetectorPlugin(registry, detectorConfig.Name, detectorConfig.Args, rfh)

	return detector
}

func convertPolicyTrigger(policyTrigger *reschedulerconfig.PolicyTrigger) *api.PolicyTrigger {
	if policyTrigger == nil {
		return nil
	}
	trigger := &api.PolicyTrigger{}
	if policyTrigger.Period != nil {
		period, err := time.ParseDuration(*policyTrigger.Period)
		if err != nil {
			klog.Errorf("failed to parse duration %s: %v", *policyTrigger.Period, err)
		}
		trigger.Period = &period
	}
	if policyTrigger.Port != nil {
		port := *policyTrigger.Port
		trigger.Port = &port
	}
	if policyTrigger.Signal != nil {
		signal := *policyTrigger.Signal
		trigger.Signal = &signal
	}
	for _, cronjob := range policyTrigger.CronjobTriggers {
		trigger.CronjobTriggers = append(trigger.CronjobTriggers, api.CronjobTrigger{Time: cronjob.Time})
	}
	return trigger
}

func newAlgorithmPlugin(registry registry.Registry, pluginName string, args apiruntime.RawExtension, rfh framework.ReschedulerFrameworkHandle) api.AlgorithmPlugin {
	plugin, err := registry[pluginName](args.Object, rfh)
	if err != nil {
		klog.Errorf("failed to new algorithm plugin %s: %v", pluginName, err)
		return nil
	}
	if pl, ok := plugin.(api.AlgorithmPlugin); ok {
		return pl
	}
	klog.Errorf("failed to new algorithm plugin %s", pluginName)
	return nil
}

func newDetectorPlugin(registry registry.Registry, pluginName string, args apiruntime.RawExtension, fh framework.ReschedulerFrameworkHandle) api.DetectorPlugin {
	plugin, err := registry[pluginName](args.Object, fh)
	if err != nil {
		klog.Errorf("failed to new detector plugin %s: %v", pluginName, err)
		return nil
	}
	if pl, ok := plugin.(api.DetectorPlugin); ok {
		return pl
	}
	klog.Errorf("failed to new detector plugin %s", pluginName)
	return nil
}
