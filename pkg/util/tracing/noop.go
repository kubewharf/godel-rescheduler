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

	"github.com/opentracing/opentracing-go"
)

var GlobalNoopTracer tracer = &NoopTracer{
	tracer: opentracing.NoopTracer{},
}

type NoopTracer struct {
	tracer opentracing.Tracer
}

func (tracer *NoopTracer) Init(componentName, idc, cluster string) error {
	return nil
}

func (tracer *NoopTracer) StartSpan(ctx context.Context, spanType, spanName string, ctxValue string, spanRef opentracing.SpanReferenceType, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	return tracer.tracer.StartSpan(spanName, opts...), ctx
}

func (tracer *NoopTracer) InjectContext(ctx context.Context, span opentracing.Span, ctxValue string) (string, error) {
	return ctxValue, nil
}

func (tracer *NoopTracer) StartSpanWithContextMap(ctx context.Context, spanType, spanName string, ctxMap map[string]string, spanRef opentracing.SpanReferenceType, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	return tracer.tracer.StartSpan(spanName, opts...), ctx
}

func (tracer *NoopTracer) InjectContextMap(ctx context.Context, span opentracing.Span, ctxMap map[string]string) (map[string]string, error) {
	return ctxMap, nil
}

func (tracer *NoopTracer) Close() error {
	return nil
}
