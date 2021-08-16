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
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"example.com/demo/v2/instrumented/helper"
)

var tracer = otel.Tracer("mux-server")

func main() {
	shutdown := helper.InitTracer("demo-server")
	defer shutdown()

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("my-server"))
	r.HandleFunc("/users/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Handling request: %q\n", r.RequestURI)
		for k, v := range r.Header {
			fmt.Printf("  %q => %q\n", k, v)
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
		return "otelmux tester"
	}
	return "unknown"
}
