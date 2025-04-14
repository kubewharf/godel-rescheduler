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
	"encoding/json"
	"reflect"
	"strings"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/podplugins"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	godelscheduler "github.com/kubewharf/godel-scheduler/pkg/scheduler"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	schedulerconfig "github.com/kubewharf/godel-scheduler/pkg/scheduler/apis/config"
	schedulerframework "github.com/kubewharf/godel-scheduler/pkg/scheduler/framework"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
)

type schedulerOptions struct {
	defaultProfile     *config.GodelSchedulerProfile
	subClusterProfiles map[string]*config.GodelSchedulerProfile

	subClusterKey string
}

// Option configures a Scheduler
type SchedulerOption func(*schedulerOptions)

func WithDefaultProfile(profile *config.GodelSchedulerProfile) SchedulerOption {
	return func(o *schedulerOptions) {
		o.defaultProfile = profile
	}
}

func WithSubClusterProfiles(profiles []config.GodelSchedulerProfile) SchedulerOption {
	return func(o *schedulerOptions) {
		subClusterProfiles := make(map[string]*config.GodelSchedulerProfile, 0)
		for i := range profiles {
			subClusterProfiles[profiles[i].SubClusterName] = &profiles[i]
		}
		o.subClusterProfiles = subClusterProfiles
	}
}

func WithSubClusterKey(key string) SchedulerOption {
	return func(o *schedulerOptions) {
		o.subClusterKey = key
	}
}

var defaultSchedulerOptions = schedulerOptions{
	subClusterKey: config.DefaultSubClusterKey,
}

func renderSchedulerOptions(opts ...SchedulerOption) schedulerOptions {
	options := defaultSchedulerOptions
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

type subClusterConfig struct {
	BasePlugins   framework.PluginCollectionSet
	PluginConfigs []schedulerconfig.PluginConfig
}

func (c *subClusterConfig) complete(profile *schedulerconfig.GodelSchedulerProfile) {
	if profile == nil {
		return
	}

	if profile.BasePluginsForKubelet != nil || profile.BasePluginsForNM != nil {
		c.BasePlugins = renderBasePlugin(godelscheduler.NewBasePlugins(), profile.BasePluginsForKubelet, profile.BasePluginsForNM)
	}
	if profile.PluginConfigs != nil {
		c.PluginConfigs = profile.PluginConfigs
	}
}

// String by JSON format. This content can be identified on `https://jsonformatter.curiousconcept.com/#`
func (c *subClusterConfig) String() string {
	bytes, _ := json.Marshal(c)
	return strings.Replace(string(bytes), "\"", "", -1)
}

func (c *subClusterConfig) Equal(other *subClusterConfig) bool {
	return reflect.DeepEqual(c, other)
}

func newDefaultSubClusterConfig(profile *schedulerconfig.GodelSchedulerProfile) *subClusterConfig {
	c := &subClusterConfig{
		BasePlugins:   godelscheduler.NewBasePlugins(),
		PluginConfigs: []schedulerconfig.PluginConfig{},
	}
	c.complete(profile)
	return c
}

func newSubClusterConfigFromDefaultConfig(profile *schedulerconfig.GodelSchedulerProfile, defaultConfig *subClusterConfig) *subClusterConfig {
	c := &subClusterConfig{
		BasePlugins:   defaultConfig.BasePlugins,
		PluginConfigs: defaultConfig.PluginConfigs,
	}

	c.complete(profile)
	return c
}

// renderBasePlugin sets the list of base plugins for pods should run
func renderBasePlugin(pluginCollection framework.PluginCollectionSet, baseKubeletPlugins, baseNMPlugins *config.Plugins) framework.PluginCollectionSet {
	getPlugins := func(plugins *config.Plugins, pluginCollection *framework.PluginCollection) {
		if plugins == nil {
			return
		}
		if plugins.Filter != nil {
			for _, plugin := range plugins.Filter.Plugins {
				pluginCollection.Filters = append(pluginCollection.Filters, framework.NewPluginSpec(plugin.Name))
			}
		}
		if plugins.Score != nil {
			for _, plugin := range plugins.Score.Plugins {
				if plugin.Weight == 0 {
					plugin.Weight = framework.DefaultPluginWeight
				}
				pluginCollection.Scores = append(pluginCollection.Scores, framework.NewPluginSpecWithWeight(plugin.Name, plugin.Weight))
			}
		}
		if plugins.VictimSearching != nil {
			pluginCollection.Searchings = make([]*framework.VictimSearchingPluginCollectionSpec, len(plugins.VictimSearching.PluginCollections))
			for i, plugin := range plugins.VictimSearching.PluginCollections {
				pluginCollection.Searchings[i] = framework.NewVictimSearchingPluginCollectionSpec(plugin.Plugins, plugin.EnableQuickPass, plugin.ForceQuickPass, plugin.RejectNotSure)
			}
		}
		if plugins.Sorting != nil {
			for _, plugin := range plugins.Sorting.Plugins {
				pluginCollection.Sortings = append(pluginCollection.Sortings, framework.NewPluginSpec(plugin.Name))
			}
		}
	}

	getPlugins(baseKubeletPlugins, pluginCollection[string(podutil.Kubelet)])
	getPlugins(baseNMPlugins, pluginCollection[string(podutil.NodeManager)])
	return pluginCollection
}

// ------------------------------- same topology filter plugin -------------------------------

func addSameTopologyFilterPlugin(config *subClusterConfig, registry schedulerframework.Registry) {
	registry[podplugins.SameTopologyPluginName] = podplugins.NewSamaTopologyPlugin
	appendToCollection := func(collection *framework.PluginCollection) {
		collection.Filters = append(collection.Filters, framework.NewPluginSpec(podplugins.SameTopologyPluginName))
	}
	appendToCollection(config.BasePlugins[string(podutil.Kubelet)])
	appendToCollection(config.BasePlugins[string(podutil.NodeManager)])
}
