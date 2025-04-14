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
	"io"

	"github.com/opentracing/opentracing-go"
)

const (
	DefaultSpanType = "GodelRescheduler"

	PodTag     = "pod"
	ClusterTag = "cluster"
	IDCTag     = "idc"
	ResultTag  = "result"
)

type TracerOption string

const (
	Noop TracerOption = "noop"
)

type (
	TraceContextVal string
	TraceContextMap map[string]string
)

var ContextError = fmt.Errorf("unsupported span context")

type tracer interface {
	Init(componentName, idc, cluster string) error
	StartSpan(ctx context.Context, spanType, spanName string, ctxValue string, spanRef opentracing.SpanReferenceType, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context)
	InjectContext(ctx context.Context, span opentracing.Span, ctxValue string) (string, error)
	StartSpanWithContextMap(ctx context.Context, spanType, spanName string, ctxMap map[string]string, spanRef opentracing.SpanReferenceType, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context)
	InjectContextMap(ctx context.Context, span opentracing.Span, ctxMap map[string]string) (map[string]string, error)
	Close() error
}

func NewTracer(tracerOption TracerOption, componentName, idc, cluster string) io.Closer {
	globalTracer = provider[tracerOption]
	if globalTracer == nil {
		globalTracer = provider[Noop]
	}
	if err := globalTracer.Init(componentName, idc, cluster); err != nil {
		globalTracer = provider[Noop]
	}
	return globalTracer
}

var (
	provider     = map[TracerOption]tracer{}
	globalTracer tracer
)

func init() {
	provider[Noop] = GlobalNoopTracer
}

func StartSpan(ctx context.Context, spanType, spanName string, ctxValue string, spanRef opentracing.SpanReferenceType, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	if globalTracer == nil {
		globalTracer = provider[Noop]
	}
	return globalTracer.StartSpan(ctx, spanType, spanName, ctxValue, spanRef, opts...)
}

func StartSpanWithContextMap(ctx context.Context, spanType, spanName string, ctxMap map[string]string, spanRef opentracing.SpanReferenceType, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	if globalTracer == nil {
		globalTracer = provider[Noop]
	}
	return globalTracer.StartSpanWithContextMap(ctx, spanType, spanName, ctxMap, spanRef, opts...)
}

func InjectContext(ctx context.Context, span opentracing.Span, ctxValue string) (string, error) {
	if globalTracer == nil {
		globalTracer = provider[Noop]
	}
	return globalTracer.InjectContext(ctx, span, ctxValue)
}

func InjectContextMap(ctx context.Context, span opentracing.Span, ctxMap map[string]string) (map[string]string, error) {
	if globalTracer == nil {
		globalTracer = provider[Noop]
	}
	return globalTracer.InjectContextMap(ctx, span, ctxMap)
}
