// Copyright 2018, OpenCensus Authors
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

package mongowrapper

import (
	"context"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sync"
	"time"
)

const (
	TracerName = "go.mongodb.org/mongo-driver"
)

var (
	keyMethod = key.New("method")
	keyStatus = key.New("status")
	keyError  = key.New("error")
)

type spanWithMetrics struct {
	startTime time.Time
	method    string
	lastErr   error
	span      trace.Span
	endOnce   sync.Once
}

func roundtripTrackingSpan(ctx context.Context, methodName string, traceOpts ...trace.SpanOption) (context.Context, *spanWithMetrics) {
	tracer := global.TraceProvider().Tracer(TracerName)
	traceOpts = append(traceOpts,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	ctx, span := tracer.Start(ctx, methodName, traceOpts...)
	return ctx, &spanWithMetrics{span: span, startTime: time.Now(), method: methodName}
}

func (swm *spanWithMetrics) setError(err error) {
	if err != nil {
		if grpcErr, ok := status.FromError(err); ok {
			swm.span.SetStatus(grpcErr.Code())
		} else {
			swm.span.SetStatus(codes.Unknown)
		}
	}
	swm.lastErr = err
}

func (swm *spanWithMetrics) end(ctx context.Context) {
	swm.endOnce.Do(func() {
		swm.span.SetAttributes(keyMethod.String(swm.method))

		if err := swm.lastErr; err == nil {
			swm.span.SetAttributes(keyStatus.String(codes.OK.String()))
		} else {
			swm.span.SetAttributes(keyError.String(err.Error()))
		}

		swm.span.End(
			trace.WithEndTime(time.Now()),
		)
	})
}
