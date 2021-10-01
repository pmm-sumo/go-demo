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

package main

import (
	"context"
	"example.com/demo/v2/instrumented/helper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"time"
)

var monoTracer = otel.Tracer("monolithic-tracer")

func main() {
	shutdown := helper.InitTracer("monolithic-demo")
	defer shutdown()

	ctx := context.Background()
	uiPost(ctx)
}

func uiPost(ctx context.Context) {
	newCtx, span := monoTracer.Start(ctx, "HTTP POST /api-endpoint",
		oteltrace.WithAttributes(attribute.String("session-id", "abcde1234"),
		attribute.String("service.name", "demo-frontend")))
	time.Sleep(time.Duration(30) * time.Millisecond)
	backendHandle(newCtx)
	time.Sleep(time.Duration(10) * time.Millisecond)
	defer span.End()
}

func backendHandle(ctx context.Context) {
	newCtx, span := monoTracer.Start(ctx, "/api-endpoint",
		oteltrace.WithAttributes(attribute.String("session-id", "abcde1234"),
		attribute.String("service.name", "backend-api")))
	time.Sleep(time.Duration(80) * time.Millisecond)
	repository(newCtx)
	time.Sleep(time.Duration(10) * time.Millisecond)
	defer span.End()
}

func repository(ctx context.Context) {
	newCtx, span := monoTracer.Start(ctx, "CallQuery",
		oteltrace.WithAttributes(attribute.String("session-id", "abcde1234"),
		attribute.String("query-id", "get-user-details"),
		attribute.String("service.name", "backend-query-repository")))
	time.Sleep(time.Duration(50) * time.Millisecond)
	db(newCtx)
	time.Sleep(time.Duration(30) * time.Millisecond)
	defer span.End()
}

func db(ctx context.Context) {
	_, span := monoTracer.Start(ctx, "SELECT x,y,z FROM abc WHERE user_id = ?",
		oteltrace.WithAttributes(attribute.String("session-id", "abcde1234"),
		attribute.String("service.name", "db")))
	time.Sleep(time.Duration(350) * time.Millisecond)
	defer span.End()
}


