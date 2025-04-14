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

package options

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/config"
	reschedulerconfig "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"

	"github.com/google/go-cmp/cmp"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilpointer "k8s.io/utils/ptr"
)

func TestLoadSchedulerFileV1beta1(t *testing.T) {
	ops, err := NewOptions()
	if err != nil {
		t.Error(err)
	}
	ops.SecureServing.BindPort = 0
	ops.CombinedInsecureServing.BindPort = 0

	fileName := "../../../../test/static/rescheduler_config.yaml"
	replaceFileName := "../../../../test/static/scheduler_config_v1beta1_preemption_default_temp.yaml"
	if err := replaceFile(fileName, replaceFileName, "{{BindPort}}", "11253"); err != nil {
		t.Error(err)
	}

	ops.ReschedulerConfigFile = replaceFileName
	ops.SchedulerConfigFile = "../../../../test/static/scheduler_config_v1beta1.yaml"
	cfg := &config.Config{}
	if err := ops.ApplyTo(cfg); err != nil {
		t.Errorf("fail to apply config: %v", err)
	}

	os.Remove(replaceFileName)

	// DefaultProfile
	{
		expectedProfile := &schedulerconfig.GodelSchedulerProfile{
			SubClusterName:                    "",
			PercentageOfNodesToScore:          utilpointer.To(int32(0)),
			IncreasedPercentageOfNodesToScore: utilpointer.To(int32(0)),
			DisablePreemption:                 utilpointer.To(true),
			BlockQueue:                        utilpointer.To(false),
			AttemptImpactFactorOnPriority:     utilpointer.To(float64(10)),
			UnitInitialBackoffSeconds:         utilpointer.To(int64(10)),
			UnitMaxBackoffSeconds:             utilpointer.To(int64(300)),
			MaxWaitingDeletionDuration:        120,
			BasePluginsForKubelet: &schedulerconfig.Plugins{
				Filter: &schedulerconfig.PluginSet{
					Plugins: []schedulerconfig.Plugin{
						{
							Name: "PodLauncher",
						}, {
							Name: "NodeUnschedulable",
						}, {
							Name: "NodeResourcesFit",
						}, {
							Name: "NodePorts",
						}, {
							Name: "VolumeBinding",
						}, {
							Name: "NodeAffinity",
						}, {
							Name: "TaintToleration",
						},
					},
				},
			},
			PluginConfigs: []schedulerconfig.PluginConfig{
				{
					Name: "NodeResourcesMostAllocated",
					Args: runtime.RawExtension{
						Raw: []uint8{123, 34, 114, 101, 115, 111, 117, 114, 99, 101, 115, 34, 58, 91, 123, 34, 110, 97, 109, 101, 34, 58, 34, 110, 118, 105, 100, 105, 97, 46, 99, 111, 109, 47, 103, 112, 117, 34, 44, 34, 119, 101, 105, 103, 104, 116, 34, 58, 49, 48, 125, 44, 123, 34, 110, 97, 109, 101, 34, 58, 34, 99, 112, 117, 34, 44, 34, 119, 101, 105, 103, 104, 116, 34, 58, 49, 125, 44, 123, 34, 110, 97, 109, 101, 34, 58, 34, 109, 101, 109, 111, 114, 121, 34, 44, 34, 119, 101, 105, 103, 104, 116, 34, 58, 49, 125, 93, 125},
						Object: &schedulerconfig.NodeResourcesMostAllocatedArgs{
							Resources: []schedulerconfig.ResourceSpec{
								{
									Name:   "nvidia.com/gpu",
									Weight: 10,
								}, {
									Name:   "cpu",
									Weight: 1,
								}, {
									Name:   "memory",
									Weight: 1,
								},
							},
						},
					},
				},
			},
		}

		profile := cfg.SchedulerConfig.DefaultProfile
		if diff := cmp.Diff(expectedProfile, profile); len(diff) > 0 {
			t.Errorf("defaultProfile got diff: %s", diff)
		}
	}

	// SubClusterProfiles: yarn
	{
		expectedProfile := &schedulerconfig.GodelSchedulerProfile{
			SubClusterName:         "yarn",
			CandidatesSelectPolicy: utilpointer.To(schedulerconfig.CandidateSelectPolicyBetter),
			BetterSelectPolicies: &schedulerconfig.StringSlice{
				schedulerconfig.BetterPreemptionPolicyAscending,
				schedulerconfig.BetterPreemptionPolicyDichotomy,
			},
			BasePluginsForKubelet: &schedulerconfig.Plugins{
				Filter: &schedulerconfig.PluginSet{
					Plugins: []schedulerconfig.Plugin{
						{
							Name: "PodLauncher",
						}, {
							Name: "NodeUnschedulable",
						}, {
							Name: "NodeResourcesFit",
						}, {
							Name: "NodePorts",
						}, {
							Name: "VolumeBinding",
						}, {
							Name: "NodeAffinity",
						}, {
							Name: "TaintToleration",
						},
					},
				},
			},
		}

		profile := getSubClusterProfile(cfg.SchedulerConfig, "yarn")
		if diff := cmp.Diff(expectedProfile, profile); len(diff) > 0 {
			t.Errorf("subCluster 1 got diff: %s", diff)
		}
	}
}

func getSubClusterProfile(compomentConfig schedulerconfig.GodelSchedulerConfiguration, subCluster string) *schedulerconfig.GodelSchedulerProfile {
	var ret *schedulerconfig.GodelSchedulerProfile
	for _, p := range compomentConfig.SubClusterProfiles {
		if p.SubClusterName == subCluster {
			if ret == nil {
				ret = &p
			} else {
				panic("Duplicate subcluster naming")
			}
		}
	}
	return ret
}

func replaceFile(fileName, replaceFileName, oldStr, newStr string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	contentStr := string(content)
	contentStr = strings.ReplaceAll(contentStr, oldStr, newStr)
	err = ioutil.WriteFile(replaceFileName, []byte(contentStr), 0o644)
	if err != nil {
		return err
	}
	return nil
}

func TestLoadReschedulerFile(t *testing.T) {
	ops, err := NewOptions()
	if err != nil {
		t.Error(err)
	}
	ops.SecureServing.BindPort = 13651
	ops.CombinedInsecureServing.BindPort = 13551

	fileName := "../../../../test/static/rescheduler_config.yaml"
	replaceFileName := "../../../../test/static/rescheduler_config_temp.yaml"
	if err := replaceFile(fileName, replaceFileName, "{{BindPort}}", "11254"); err != nil {
		t.Error(err)
	}

	ops.ReschedulerConfigFile = replaceFileName
	cfg := &config.Config{}
	if err := ops.ApplyTo(cfg); err != nil {
		t.Errorf("fail to apply config: %v", err)
	}

	os.Remove(replaceFileName)

	// DefaultProfile
	{
		expectedProfile := &reschedulerconfig.ReschedulerProfile{
			ReschedulerPolicyConfigs: []reschedulerconfig.ReschedulerPolicy{
				{
					PolicyTrigger: &reschedulerconfig.PolicyTrigger{
						Signal: utilpointer.To(12),
					},
					Detector: reschedulerconfig.DetectorConfig{
						Name: "BinPacking",
						Args: runtime.RawExtension{
							Raw: []uint8{123, 34, 114, 101, 115, 111, 117, 114, 99, 101, 73, 116, 101, 109, 115, 34, 58, 91, 123, 34, 114, 101, 115, 111, 117, 114, 99, 101, 34, 58, 34, 99, 112, 117, 34, 44, 34, 119, 101, 105, 103, 104, 116, 34, 58, 49, 125, 93, 44, 34, 116, 104, 114, 101, 115, 104, 111, 108, 100, 80, 101, 114, 99, 101, 110, 116, 97, 103, 101, 34, 58, 48, 46, 53, 125},
							Object: &reschedulerconfig.BinPackingArgs{
								ResourceItems: []reschedulerconfig.ResourceItem{
									{
										Resource: "cpu",
										Weight:   1,
									},
								},
								ThresholdPercentage: 0.5,
							},
						},
						DetectorTrigger: &reschedulerconfig.PolicyTrigger{
							Signal: utilpointer.To(12),
						},
					},
					Algorithm: reschedulerconfig.AlgorithmConfig{
						Name: "BinPacking",
						Args: runtime.RawExtension{
							Raw:    nil,
							Object: &reschedulerconfig.BinPackingArgs{},
						},
						CheckPolicies: []string{"Protection"},
					},
					FailureRatioThreshold: utilpointer.To(0.3),
				},
				{
					Detector: reschedulerconfig.DetectorConfig{
						Name: "Protection",
						Args: runtime.RawExtension{},
					},
					Algorithm: reschedulerconfig.AlgorithmConfig{
						Name: "Protection",
						Args: runtime.RawExtension{},
					},
				},
				{
					Detector: reschedulerconfig.DetectorConfig{
						Name: "Throttle",
						Args: runtime.RawExtension{
							Raw:    nil,
							Object: &reschedulerconfig.ThrottleArgs{},
						},
					},
					Algorithm: reschedulerconfig.AlgorithmConfig{
						Name: "Throttle",
						Args: runtime.RawExtension{
							Raw: []uint8{123, 34, 116, 104, 114, 111, 116, 116, 108, 101, 73, 116, 101, 109, 115, 34, 58, 91, 123, 34, 108, 97, 98, 101, 108, 83, 101, 108, 101, 99, 116, 111, 114, 75, 101, 121, 34, 58, 34, 112, 115, 109, 34, 44, 34, 116, 104, 114, 111, 116, 116, 108, 101, 68, 117, 114, 97, 116, 105, 111, 110, 34, 58, 34, 53, 109, 34, 44, 34, 116, 104, 114, 111, 116, 116, 108, 101, 86, 97, 108, 117, 101, 34, 58, 49, 125, 93, 125},
							Object: &reschedulerconfig.ThrottleArgs{
								ThrottleItems: []reschedulerconfig.ThrottleItem{
									{
										LabelSelectorKey: "psm",
										ThrottleDuration: metav1.Duration{Duration: time.Minute * 5},
										ThrottleValue:    1,
									},
								},
							},
						},
					},
				},
			},
		}
		profile := cfg.ReschedulerConfig.Profile
		if diff := cmp.Diff(expectedProfile.ReschedulerPolicyConfigs, profile.ReschedulerPolicyConfigs); len(diff) > 0 {
			t.Errorf("defaultProfile got diff: %s", diff)
		}
	}
}
