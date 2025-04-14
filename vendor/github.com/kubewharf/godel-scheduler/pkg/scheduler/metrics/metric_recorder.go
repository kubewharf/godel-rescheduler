/*
Copyright 2023 The Godel Scheduler Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	pkgmetrics "github.com/kubewharf/godel-scheduler/pkg/common/metrics"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
)

var _ pkgmetrics.MetricRecorder = &PendingUnitsRecorder{}

// PendingUnitsRecorder is an implementation of MetricRecorder
type PendingUnitsRecorder struct {
	queue string
}

func newPendingUnitsRecorder(queue string) *PendingUnitsRecorder {
	return &PendingUnitsRecorder{
		queue: queue,
	}
}

// NewPendingUnitsRecorder return PendingUnitsRecorder about given queue
func NewPendingUnitsRecorder(queue string) *PendingUnitsRecorder {
	return newPendingUnitsRecorder(queue)
}

// Inc increases the pod and unit metrics, in an atomic way
func (r *PendingUnitsRecorder) Inc(obj interface{}) {
	if obj != nil {
		if storedUnit, ok := obj.(framework.StoredUnit); ok && storedUnit != nil {
			if storedUnit.NumPods() <= 0 {
				return
			}

			observableUnit, ok := storedUnit.(framework.ObservableUnit)
			if !ok {
				return
			}

			unitProperty := observableUnit.GetUnitProperty()
			if unitProperty == nil {
				return
			}

			podProperty := unitProperty.GetPodProperty()
			if podProperty == nil {
				return
			}

			PendingPodsAdd(podProperty, r.queue, float64(storedUnit.NumPods()))
			PendingUnitsInc(unitProperty, r.queue)

			for _, info := range storedUnit.GetPods() {
				info.UpdateQueueStage(r.queue)
			}
		}
	}
}

// Dec decreases the pod and unit metrics, in an atomic way
func (r *PendingUnitsRecorder) Dec(obj interface{}) {
	if obj != nil {
		if storedUnit, ok := obj.(framework.StoredUnit); ok && storedUnit != nil {
			if storedUnit.NumPods() <= 0 {
				return
			}

			observableUnit, ok := storedUnit.(framework.ObservableUnit)
			if !ok {
				return
			}

			unitProperty := observableUnit.GetUnitProperty()
			if unitProperty == nil {
				return
			}

			podProperty := unitProperty.GetPodProperty()
			if podProperty == nil {
				return
			}

			PendingPodsAdd(podProperty, r.queue, -float64(storedUnit.NumPods()))
			PendingUnitsDec(unitProperty, r.queue)
		}
	}
}

// Clear set a metric counter to 0, in an atomic way
func (r *PendingUnitsRecorder) Clear() {
	// no-op
}

func (r *PendingUnitsRecorder) AddingLatencyInSeconds(obj interface{}, duration float64) {
	if obj != nil {
		if storedUnit, ok := obj.(framework.StoredUnit); ok && storedUnit != nil {
			if storedUnit.NumPods() <= 0 {
				return
			}

			observableUnit, ok := storedUnit.(framework.ObservableUnit)
			if !ok {
				return
			}

			unitProperty := observableUnit.GetUnitProperty()
			if unitProperty == nil {
				return
			}

			podProperty := unitProperty.GetPodProperty()
			if podProperty == nil {
				return
			}

			QueueSortingLatencyObserve(podProperty, r.queue, duration)
		}
	}
}
