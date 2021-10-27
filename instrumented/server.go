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
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	oteltrace "go.opentelemetry.io/otel/trace"

	"example.com/demo/v2/instrumented/helper"
)

var tracer = otel.Tracer("mux-server")
var meter = global.Meter("demo-meter")

func main() {
	shutdown := helper.InitTracer("demo-server")
	defer shutdown()

	metricShutdown := helper.InitMeter()
	defer metricShutdown()

	// Ensure it behaves like TTY is disabled
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	userCounter := otelmetric.Must(meter).NewInt64Counter("users_req_count",
		otelmetric.WithDescription("Number of requests to /users"))

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("my-server"))
	r.HandleFunc("/users/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		userCounter.Add(context.Background(), 1)
		logrus.Infof("Handling request: %q\n", r.RequestURI)
		for k, v := range r.Header {
			logrus.Infof("  %q => %q\n", k, v)
		}
		vars := mux.Vars(r)
		id := vars["id"]
		name := getUser(r.Context(), id)
		reply := fmt.Sprintf("user %s (id %s)\n", name, id)
		_, _ = w.Write(([]byte)(reply))
	})
	http.Handle("/", r)
	_ = http.ListenAndServe(":8080", r)
}

func getUser(ctx context.Context, id string) string {
	_, span := tracer.Start(ctx, "getUser", oteltrace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	if id == "123" {
		logrus.WithFields(helper.LogrusFields(span)).Infof("Handling User ID: %s", id)
		return "otelmux tester"
	} else {
		span.SetStatus(codes.Error, "No user found")
		logrus.WithFields(helper.LogrusFields(span)).Warnf("User ID: %s not found", id)
	}
	return "unknown"
}
