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
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/config"
	"github.com/kubewharf/godel-rescheduler/cmd/godel-rescheduler/app/options"

	"github.com/google/go-cmp/cmp"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/plugins/nodeports"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/plugins/coscheduling"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/plugins/nodeaffinity"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/plugins/noderesources"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/plugins/nodeunschedulable"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/plugins/podlauncher"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/plugins/tainttoleration"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/plugins/volumebinding"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/preemption-plugins/searching/newlystartedprotectionchecker"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/preemption-plugins/searching/pdbchecker"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/preemption-plugins/searching/podlauncherchecker"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/preemption-plugins/searching/preemptibilitychecker"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/preemption-plugins/searching/priorityvaluechecker"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/preemption-plugins/sorting/priority"
	starttime "github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/preemption-plugins/sorting/start_time"
	victimscount "github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/preemption-plugins/sorting/victims_count"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestLoadAndRenderSchedulerFileV1beta1(t *testing.T) {
	ops, err := options.NewOptions()
	if err != nil {
		t.Error(err)
	}
	ops.SecureServing.BindPort = 0
	ops.CombinedInsecureServing.BindPort = 0

	fileName := "../../test/static/rescheduler_config.yaml"
	replaceFileName := "../../test/static/scheduler_config_v1beta1_preemption_default_temp.yaml"
	if err := replaceFile(fileName, replaceFileName, "{{BindPort}}", "11252"); err != nil {
		t.Error(err)
	}

	ops.ReschedulerConfigFile = replaceFileName
	ops.SchedulerConfigFile = "../../test/static/scheduler_config_v1beta1.yaml"
	cfg := &config.Config{}
	if err := ops.ApplyTo(cfg); err != nil {
		t.Errorf("fail to apply config: %v", err)
	}

	os.Remove(replaceFileName)

	defaultProfile := newDefaultSubClusterConfig(cfg.SchedulerConfig.DefaultProfile)
	// DefaultProfile
	{
		expectedProfile := &subClusterConfig{
			BasePlugins: framework.PluginCollectionSet{
				string(podutil.Kubelet): {
					Filters: []*framework.PluginSpec{
						framework.NewPluginSpec(podlauncher.Name),
						framework.NewPluginSpec(coscheduling.Name),
						framework.NewPluginSpec(nodeunschedulable.Name),
						framework.NewPluginSpec(noderesources.FitName),
						framework.NewPluginSpec(nodeports.Name),
						framework.NewPluginSpec(volumebinding.Name),
						framework.NewPluginSpec(nodeaffinity.Name),
						framework.NewPluginSpec(tainttoleration.Name),
						framework.NewPluginSpec(podlauncher.Name),
						framework.NewPluginSpec(nodeunschedulable.Name),
						framework.NewPluginSpec(noderesources.FitName),
						framework.NewPluginSpec(nodeports.Name),
						framework.NewPluginSpec(volumebinding.Name),
						framework.NewPluginSpec(nodeaffinity.Name),
						framework.NewPluginSpec(tainttoleration.Name),
					},
					Searchings: []*framework.VictimSearchingPluginCollectionSpec{
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: podlauncherchecker.PodLauncherCheckerName},
							},
							false,
							false,
							false,
						),
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: preemptibilitychecker.PreemptibilityCheckerName},
							},
							false,
							false,
							false,
						),
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: newlystartedprotectionchecker.NewlyStartedProtectionCheckerName},
							},
							false,
							false,
							false,
						),
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: priorityvaluechecker.PriorityValueCheckerName},
							},
							false,
							false,
							true,
						),
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: pdbchecker.PDBCheckerName},
							},
							false,
							false,
							false,
						),
					},
					Sortings: []*framework.PluginSpec{
						framework.NewPluginSpec(priority.MinHighestPriorityName),
						framework.NewPluginSpec(priority.MinPrioritySumName),
						framework.NewPluginSpec(victimscount.LeastVictimsName),
						framework.NewPluginSpec(starttime.LatestEarliestStartTimeName),
					},
				},
				string(podutil.NodeManager): {
					Filters: []*framework.PluginSpec{
						framework.NewPluginSpec(podlauncher.Name),
						framework.NewPluginSpec(coscheduling.Name),
						framework.NewPluginSpec(nodeunschedulable.Name),
						framework.NewPluginSpec(noderesources.FitName),
						framework.NewPluginSpec(nodeports.Name),
						framework.NewPluginSpec(volumebinding.Name),
						framework.NewPluginSpec(nodeaffinity.Name),
						framework.NewPluginSpec(tainttoleration.Name),
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
		if diff := cmp.Diff(expectedProfile, defaultProfile); len(diff) > 0 {
			t.Errorf("defaultProfile got diff: %s", diff)
		}
		t.Log(expectedProfile.String())
	}

	// SubClusterProfiles: subCluster priorityqueue
	// Fields that are not set should use defaultProfile.
	{
		name := "yarn"
		subClusterProfile := newSubClusterConfigFromDefaultConfig(getSubClusterProfile(cfg.SchedulerConfig, name), defaultProfile)
		expectedProfile := &subClusterConfig{
			BasePlugins: framework.PluginCollectionSet{
				string(podutil.Kubelet): {
					Filters: []*framework.PluginSpec{
						framework.NewPluginSpec(podlauncher.Name),
						framework.NewPluginSpec(coscheduling.Name),
						framework.NewPluginSpec(nodeunschedulable.Name),
						framework.NewPluginSpec(noderesources.FitName),
						framework.NewPluginSpec(nodeports.Name),
						framework.NewPluginSpec(volumebinding.Name),
						framework.NewPluginSpec(nodeaffinity.Name),
						framework.NewPluginSpec(tainttoleration.Name),
						framework.NewPluginSpec(podlauncher.Name),
						framework.NewPluginSpec(nodeunschedulable.Name),
						framework.NewPluginSpec(noderesources.FitName),
						framework.NewPluginSpec(nodeports.Name),
						framework.NewPluginSpec(volumebinding.Name),
						framework.NewPluginSpec(nodeaffinity.Name),
						framework.NewPluginSpec(tainttoleration.Name),
					},
					Searchings: []*framework.VictimSearchingPluginCollectionSpec{
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: podlauncherchecker.PodLauncherCheckerName},
							},
							false,
							false,
							false,
						),
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: preemptibilitychecker.PreemptibilityCheckerName},
							},
							false,
							false,
							false,
						),
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: newlystartedprotectionchecker.NewlyStartedProtectionCheckerName},
							},
							false,
							false,
							false,
						),
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: priorityvaluechecker.PriorityValueCheckerName},
							},
							false,
							false,
							true,
						),
						framework.NewVictimSearchingPluginCollectionSpec(
							[]schedulerconfig.Plugin{
								{Name: pdbchecker.PDBCheckerName},
							},
							false,
							false,
							false,
						),
					},
					Sortings: []*framework.PluginSpec{
						framework.NewPluginSpec(priority.MinHighestPriorityName),
						framework.NewPluginSpec(priority.MinPrioritySumName),
						framework.NewPluginSpec(victimscount.LeastVictimsName),
						framework.NewPluginSpec(starttime.LatestEarliestStartTimeName),
					},
				},
				string(podutil.NodeManager): {
					Filters: []*framework.PluginSpec{
						framework.NewPluginSpec(podlauncher.Name),
						framework.NewPluginSpec(coscheduling.Name),
						framework.NewPluginSpec(nodeunschedulable.Name),
						framework.NewPluginSpec(noderesources.FitName),
						framework.NewPluginSpec(nodeports.Name),
						framework.NewPluginSpec(volumebinding.Name),
						framework.NewPluginSpec(nodeaffinity.Name),
						framework.NewPluginSpec(tainttoleration.Name),
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
		if diff := cmp.Diff(expectedProfile, subClusterProfile); len(diff) > 0 {
			t.Errorf("subClusterProfile got diff: %s", diff)
		}
		t.Log(expectedProfile.String())
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
