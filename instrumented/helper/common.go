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

package helper

import (
	"context"
	"log"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	zipkin "go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.9.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"

	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
)

const (
	enableZipkin = true
	enableOtlp   = false
)

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func InitTracer(serviceName string) func() {
	return initTracer(serviceName)
	//return initLogTracer()
}

func InitMeter() func() {
	return initOtlpMeter()
}

func LogrusFields(span oteltrace.Span) logrus.Fields {
	return logrus.Fields{
		"span_id":  span.SpanContext().SpanID().String(),
		"trace_id": span.SpanContext().TraceID().String(),
	}
}

func initOtlpMeter() func() {
	ctx := context.Background()
	client := otlpmetricgrpc.NewClient(otlpmetricgrpc.WithInsecure())
	exporter, err := otlpmetric.New(ctx, client)
	handleErr(err)

	pusher := controller.New(
		processor.NewFactory(
			simple.NewWithInexpensiveDistribution(),
			exporter,
		),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(2*time.Second),
	)
	global.SetMeterProvider(pusher)

	if err := pusher.Start(ctx); err != nil {
		handleErr(err)
	}

	return func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := exporter.Shutdown(ctx); err != nil {
			handleErr(err)
		}
	}
}

func initTracer(serviceName string) func() {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	handleErr(err)

	// Create stdout exporter to be able to retrieve
	// the collected spans.
	stdOutExporter, err := stdout.New(stdout.WithPrettyPrint())
	handleErr(err)

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(stdOutExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	if enableOtlp {
		otlpTraceExporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint("localhost:4317"),
			otlptracegrpc.WithDialOption(grpc.WithBlock()),
		)
		handleErr(err)

		otlpBatcher := sdktrace.NewBatchSpanProcessor(otlpTraceExporter)
		tracerProvider.RegisterSpanProcessor(otlpBatcher)
	}

	if enableZipkin {
		zipkinExporter, err := zipkin.New("http://localhost:9411/api/v2/spans")
		handleErr(err)

		zipkinBatcher := sdktrace.NewBatchSpanProcessor(zipkinExporter)
		tracerProvider.RegisterSpanProcessor(zipkinBatcher)
	}

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		// Shutdown will flush any remaining spans and shut down the exporter.
		handleErr(tracerProvider.Shutdown(context.Background()))
	}
}

func initLogTracer() func() {
	// Create stdout exporter to be able to retrieve
	// the collected spans.
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		// Shutdown will flush any remaining spans and shut down the exporter.
		handleErr(tp.Shutdown(context.Background()))
	}
}
