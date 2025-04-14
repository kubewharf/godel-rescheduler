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

package tracing

import (
	"context"
	"fmt"

	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	"github.com/opentracing/opentracing-go"
	v1 "k8s.io/api/core/v1"
)

type TracerConfig string

const (
	JaegerConfig TracerConfig = "jaeger"
	NoopConfig   TracerConfig = "noop"
)

var InvalidTracerConfigError error = fmt.Errorf("invalidate tracer")

func ValidateTracerConfig(config TracerConfig) (TracerOption, error) {
	switch config {
	case NoopConfig:
		return Noop, nil
	default:
		return "", InvalidTracerConfigError
	}
}

// StartSpanForPod creates a span and tracing context of related pod. The new span will be based on spanCtx if it is not empty.
// If both spanCtx is empty, a root span is created for the pod and return the new tracing context as spanCtx value.
// The returned value will include the new created span and spanCtx.
// If root span is created, spanCtx from root span is returned, otherwise returned empty.
func StartSpanForPod(ctx context.Context, pod *v1.Pod, spanName, parentSpanCtx string, spanRef opentracing.SpanReferenceType, options ...opentracing.StartSpanOption) (opentracing.Span, context.Context, string) {
	podKey := podutil.GetPodKey(pod)
	opts := []opentracing.StartSpanOption{
		opentracing.Tag{
			Key:   PodTag,
			Value: podKey,
		},
	}
	opts = append(opts, options...)
	rootCtx := ""
	if parentSpanCtx == "" {
		rootSpan, spanCtx := StartSpan(ctx, DefaultSpanType, "rootSpan", "", opentracing.FollowsFromRef, opts...)
		defer func() {
			go rootSpan.Finish()
		}()

		if span, err := InjectContext(spanCtx, rootSpan, ""); err == nil {
			rootCtx = span
		}
		parentSpanCtx = rootCtx
	}

	newSpan, newCtx := StartSpan(ctx, DefaultSpanType, spanName, parentSpanCtx, spanRef, opts...)
	return newSpan, newCtx, rootCtx
}

// StartSpanForPodWithContextMap creates a span and tracing context of related pod. The new span will be based on spanCtx if it is not empty.
// If both spanCtx is empty, a root span is created for the pod and return the new tracing context as spanCtx value.
// The returned value will include the new created span and spanCtx.
// If root span is created, spanCtx from root span is returned, otherwise returned empty.
func StartSpanForPodWithContextMap(
	ctx context.Context,
	pod *v1.Pod,
	spanName string,
	parentSpanCtx map[string]string,
	spanRef opentracing.SpanReferenceType,
	options ...opentracing.StartSpanOption,
) (opentracing.Span, context.Context, map[string]string) {
	podKey := podutil.GetPodKey(pod)
	opts := []opentracing.StartSpanOption{
		opentracing.Tag{
			Key:   PodTag,
			Value: podKey,
		},
	}
	opts = append(opts, options...)
	rootCtx := make(map[string]string)
	if len(parentSpanCtx) == 0 {
		parentSpanCtx = make(map[string]string)
		rootSpan, spanCtx := StartSpanWithContextMap(ctx, DefaultSpanType, "rootSpan", parentSpanCtx, opentracing.FollowsFromRef, opts...)
		defer func() {
			go rootSpan.Finish()
		}()

		if span, err := InjectContextMap(spanCtx, rootSpan, parentSpanCtx); err == nil {
			rootCtx = span
		}
		parentSpanCtx = rootCtx
	}

	newSpan, newCtx := StartSpanWithContextMap(ctx, DefaultSpanType, spanName, parentSpanCtx, spanRef, opts...)
	return newSpan, newCtx, rootCtx
}

func GetTracingCtxForPod(pod *v1.Pod) string {
	return pod.Annotations[podutil.TraceContext]
}

func SetTracingCtxForPod(pod *v1.Pod, ctx string) {
	pod.Annotations[podutil.TraceContext] = ctx
}
