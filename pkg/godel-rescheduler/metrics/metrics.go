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

package metrics

import (
	"sync"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

const (
	// ReschedulerSubsystem - subsystem name used by scheduler
	ReschedulerSubsystem = "rescheduler"
	// ReschedulerLabel - label used by scheduler
	ReschedulerLabel = "rescheduler"

	// SuccessResult - result label value
	SuccessResult = "success"
	// FailureResult - result label value
	FailureResult = "failure"

	DetectorEvaluation           = "detector_evaluation"
	AlgorithmEvaluation          = "algorithm_evaluation"
	MovementControllerEvaluation = "movement_controller_evaluation"
)

// All the histogram based metrics have 1ms as size for the smallest bucket.
var (
	ReschedulerCacheSize = metrics.NewGaugeVec(
		&metrics.GaugeOpts{
			Subsystem:      ReschedulerSubsystem,
			Name:           "rescheduler_cache_size",
			Help:           "Number of nodes in the rescheduler cache.",
			StabilityLevel: metrics.ALPHA,
		}, []string{"type"})

	PodReschedulingStageDuration = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Subsystem:      ReschedulerSubsystem,
			Name:           "rescheduling_stage_duration_seconds",
			Help:           "rescheduling latency in each plugin",
			Buckets:        metrics.ExponentialBuckets(0.001, 2, 20),
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"operation", "plugin"},
	)

	rescheduledPodCountInAllMovements = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Subsystem:      ReschedulerSubsystem,
			Name:           "rescheduled_pod_count_in_all_movements",
			Help:           "Number of rescheduled pod in all movements.",
			StabilityLevel: metrics.ALPHA,
		}, []string{"plugin", "class", "reason"})

	rescheduledPodCountInEachMovement = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Subsystem:      ReschedulerSubsystem,
			Name:           "rescheduled_pod_count_in_each_movement",
			Help:           "Number of rescheduled pod in each movement.",
			StabilityLevel: metrics.ALPHA,
		}, []string{"plugin", "class"})

	metricsList = []metrics.Registerable{
		ReschedulerCacheSize,
		PodReschedulingStageDuration,
		rescheduledPodCountInAllMovements,
		rescheduledPodCountInEachMovement,
	}
)

var registerMetrics sync.Once

// Register all metrics.
func Register() {
	// Register the metrics.
	registerMetrics.Do(func() {
		RegisterMetrics(metricsList...)
	})
}

// RegisterMetrics registers a list of metrics.
// This function is exported because it is intended to be used by out-of-tree plugins to register their custom metrics.
func RegisterMetrics(extraMetrics ...metrics.Registerable) {
	for _, metric := range extraMetrics {
		legacyregistry.MustRegister(metric)
	}
}

// GetGather returns the gatherer. It used by test case outside current package.
func GetGather() metrics.Gatherer {
	return legacyregistry.DefaultGatherer
}
