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

package features

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
)

func init() {
	runtime.Must(utilfeature.DefaultMutableFeatureGate.Add(defaultReschedulerFeatureGates))
}

// defaultReschedulerFeatureGates consists of all known Rescheduler-specific feature keys.
// To add a new feature, define a key for it above and add it here. The features will be
// available throughout Rescheduler binaries.
var defaultReschedulerFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{}
