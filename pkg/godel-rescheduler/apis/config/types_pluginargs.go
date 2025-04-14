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

package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeResourcesExclusiveAllocatedArgs holds arguments used to configure NodeResourcesExclusiveAllocatedArgs plugin.
type BinPackingArgs struct {
	metav1.TypeMeta `json:",inline"`

	// threshold trigger of detector plugin
	ResourceItems       []ResourceItem `json:"resourceItems"`
	ThresholdPercentage float64        `json:"thresholdPercentage"`
}

type ThresholdTrigger struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type ResourceItem struct {
	Resource string `json:"resource"`
	Weight   int64  `json:"weight"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ThrottleArgs holds arguments used to configure Throttle plugin.
type ThrottleArgs struct {
	metav1.TypeMeta `json:",inline"`

	ThrottleItems []ThrottleItem `json:"throttleItems,omitempty"`
}

type ThrottleItem struct {
	LabelSelectorKey string          `json:"labelSelectorKey,omitempty"`
	ThrottleDuration metav1.Duration `json:"throttleDuration,omitempty"`
	ThrottleValue    int64           `json:"throttleValue,omitempty"`
	CanBePreempted   *bool           `json:"canBePreempted,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GroupThrottleArgs holds arguments used to configure GroupThrottle plugin.
type GroupThrottleArgs struct {
	metav1.TypeMeta `json:",inline"`

	ThrottleItems []ThrottleItem `json:"throttleItems,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PendingCheckerArgs holds arguments used to configure PendingChecker plugin.
type PendingCheckerArgs struct {
	metav1.TypeMeta `json:",inline"`

	// key is sub cluster, value is pending pods count threshold
	PendingThresholds       map[string]int `json:"pendingThresholds,omitempty"`
	DefaultPendingThreshold *int           `json:"defaultPendingThreshold,omitempty"`
}
