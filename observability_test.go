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
	"errors"
	"go.opentelemetry.io/otel/api/global"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/codes"
	"testing"
	"time"
)

func TestUnitRoundtripTrackingOperation(t *testing.T) {
	reportingPeriod := 200 * time.Millisecond

	spanDataChan := make(chan *export.SpanData, 1)
	exp := &mockExporter{spanDataChan: spanDataChan}
	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exp),
	)
	if err != nil {
		t.Error("failed to set tracing provider", err)
	}

	global.SetTraceProvider(tp)

	pausePeriod := 28 * time.Millisecond
	deadline := time.Now().Add(reportingPeriod)
	_, rts := roundtripTrackingSpan(context.Background(), "a.b.c/D.Foo")
	<-time.After(pausePeriod / 2)
	errMsg := "This is an error"
	rts.setError(errors.New(errMsg))
	<-time.After(pausePeriod / 2)
	rts.end(context.Background())

	// Verifying the spans since those don't
	// operate on a frequency.
	sd0 := <-spanDataChan
	// Comparing the name
	if g, w := sd0.Name, TracerName + "/a.b.c/D.Foo"; g != w {
		t.Errorf("SpanData.Name mismatch:: Got %q Want %q", g, w)
	}
	wantStatus := codes.Unknown
	if g, w := sd0.Status, wantStatus; g != w {
		t.Errorf("SpanData.Status mismatch:: Got %#v Want %#v", g, w)
	}
	minPeriod := pausePeriod
	gotPeriod := sd0.EndTime.Sub(sd0.StartTime)
	if gotPeriod < minPeriod {
		t.Errorf("SpanData.TimeSpent:: Got %s Want min: %s", gotPeriod, minPeriod)
	}

	wait := deadline.Sub(time.Now()) + 3*time.Millisecond
	<-time.After(wait)
}

type mockExporter struct {
	spanDataChan chan *export.SpanData
}

var _ export.SpanSyncer = (*mockExporter)(nil)

func (me *mockExporter) ExportSpan(ctx context.Context, sd *export.SpanData) {
	me.spanDataChan <- sd
}
