// Copyright The OpenTelemetry Authors
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

package propagation_test // import "go.opentelemetry.io/otel/propagation"

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func ExampleServerTimingPropagator() {
	// in your main function, or equivalent:
	p := &serverTimingPropagator{}
	otel.SetTextMapResponsePropagator(p)

	// your code would be instrumented as usual
	tr := otel.Tracer("example")
	ctx, span := tr.Start(context.Background(), "operation")
	defer span.End()

	// the library you use to handle HTTP requests would call this when sending the response back to the caller:
	hc := make(propagation.HeaderCarrier)
	otel.GetTextMapResponsePropagator().Inject(ctx, hc)

	// Output: traceresponse;desc=00-00000000000000000000000000000000-0000000000000000-00
	fmt.Println(hc.Get("Server-Timing"))
}

type serverTimingPropagator struct {
}

func (p serverTimingPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	header := carrier.Get("Server-Timing")
	if header == "" {
		return ctx
	}

	// TODO: validate the header
	desc := strings.Split(header, ";")[1]
	values := strings.Split(desc, "-")

	traceID, _ := trace.TraceIDFromHex(values[1])
	spanID, _ := trace.SpanIDFromHex(values[2])

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: 0, // TODO: properly parse this
	})

	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

func (serverTimingPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	tctx := trace.SpanContextFromContext(ctx)
	traceID := tctx.TraceID()
	spanID := tctx.SpanID()

	samplingFlag := "00"
	if tctx.IsSampled() {
		samplingFlag = "01"
	}

	header := fmt.Sprintf("%s;desc=%s-%s-%s-%s", "traceresponse", "00", traceID, spanID, samplingFlag)
	carrier.Set("Server-Timing", header)
}

func (serverTimingPropagator) Fields() []string {
	return []string{"Server-Timing"}
}
