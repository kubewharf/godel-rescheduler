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

package registry

import (
	api "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	binpacking "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/plugins/bin-packing"
	groupthrottle "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/plugins/group-throttle"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/plugins/pdbchecker"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/plugins/pendingchecker"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/plugins/protection"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/plugins/throttle"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"k8s.io/apimachinery/pkg/runtime"
)

// PluginFactory is a function that builds a plugin.
type PluginFactory = func(configuration runtime.Object, rfh api.ReschedulerFrameworkHandle) (framework.Plugin, error)

// Registry is a collection of all available plugins. The framework uses a
// registry to enable and initialize configured plugins.
// All plugins must be in the registry before initializing the framework.
type Registry map[string]PluginFactory

// NewInTreeRegistry builds the registry with all the in-tree plugins.
// A scheduler that runs out of tree plugins can register additional plugins
// through the WithFrameworkOutOfTreeRegistry option.
// For Godel Resheduler all in tree plugins are enabled
func NewInTreeRegistry() Registry {
	return Registry{
		binpacking.Name:     binpacking.New,
		protection.Name:     protection.New,
		pdbchecker.Name:     pdbchecker.New,
		throttle.Name:       throttle.New,
		groupthrottle.Name:  groupthrottle.New,
		pendingchecker.Name: pendingchecker.New,
	}
}
