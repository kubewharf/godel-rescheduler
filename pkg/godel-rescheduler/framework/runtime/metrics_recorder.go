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

package runtime

import (
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/metrics"

	k8smetrics "k8s.io/component-base/metrics"
)

// frameworkMetric is the data structure passed in the buffer channel between the main framework thread
// and the MetricsRecorder goroutine.
type frameworkMetric struct {
	metric      *k8smetrics.HistogramVec
	labelValues []string
	value       float64
}

// metricRecorder records framework metrics in a separate goroutine to avoid overhead in the critical path.
type MetricsRecorder struct {
	// bufferCh is a channel that serves as a metrics buffer before the MetricsRecorder goroutine reports it.
	bufferCh chan *frameworkMetric
	// if bufferSize is reached, incoming metrics will be discarded.
	bufferSize int
	// how often the recorder runs to flush the metrics.
	interval time.Duration

	// stopCh is used to stop the goroutine which periodically flushes metrics. It's currently only
	// used in tests.
	stopCh chan struct{}
	// isStoppedCh indicates whether the goroutine is stopped. It's used in tests only to make sure
	// the metric flushing goroutine is stopped so that tests can collect metrics for verification.
	isStoppedCh chan struct{}
}

func NewMetricsRecorder(bufferSize int, interval time.Duration) *MetricsRecorder {
	recorder := &MetricsRecorder{
		bufferCh:    make(chan *frameworkMetric, bufferSize),
		bufferSize:  bufferSize,
		interval:    interval,
		stopCh:      make(chan struct{}),
		isStoppedCh: make(chan struct{}),
	}
	go recorder.run()
	return recorder
}

// observeOperationDurationAsync observes the plugin_execution_duration_seconds metric.
// The metric will be flushed to Prometheus asynchronously.
func (r *MetricsRecorder) ObserveOperationDurationAsync(extensionPoint, pluginName string, value float64) {
	newMetric := &frameworkMetric{
		metric:      metrics.PodReschedulingStageDuration,
		labelValues: []string{extensionPoint, pluginName},
		value:       value,
	}
	select {
	case r.bufferCh <- newMetric:
	default:
	}
}

// run flushes buffered metrics into Prometheus every second.
func (r *MetricsRecorder) run() {
	for {
		select {
		case <-r.stopCh:
			close(r.isStoppedCh)
			return
		default:
		}
		r.flushMetrics()
		time.Sleep(r.interval)
	}
}

// flushMetrics tries to clean up the bufferCh by reading at most bufferSize metrics.
func (r *MetricsRecorder) flushMetrics() {
	for i := 0; i < r.bufferSize; i++ {
		select {
		case m := <-r.bufferCh:
			m.metric.WithLabelValues(m.labelValues...).Observe(m.value)
		default:
			return
		}
	}
}
