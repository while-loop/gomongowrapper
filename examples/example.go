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

package main

import (
	"context"
	"github.com/while-loop/gomongowrapper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporter/trace/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log"
)

func main() {
	// Enabling the OpenTelemetry exporter.
	// Just using Stackdriver since it has both Tracing and Metrics
	// and is easy to whip up. Add your desired one here.
	InitTracer()

	tracer := global.TraceProvider().Tracer("example-tracer")

	// Start a span like your application would start one.
	ctx, span := tracer.Start(context.Background(), "Fetch")
	defer span.End()

	// Now for the mongo connections, using the context
	// with the span in it for continuity.
	client, err := mongowrapper.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Failed to create the new client: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to open client connection: %v", err)
	}
	defer client.Disconnect(ctx)
	coll := client.Database("the_db").Collection("music")

	q := bson.M{"name": "Examples"}
	cur, err := coll.Find(ctx, q)
	if err != nil {
		log.Fatalf("Find error: %v", err)
	}

	for cur.Next(ctx) {
		elem := make(map[string]int)
		if err := cur.Decode(elem); err != nil {
			log.Printf("Decode error: %v", err)
			continue
		}
		log.Printf("Got result: %v\n", elem)
	}
	log.Print("Done iterating")

	_, err = coll.DeleteMany(ctx, q)
	if err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}
}

func InitTracer() {
	ex, err := stdout.NewExporter(stdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}
	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(ex),
	)

	global.SetTraceProvider(tp)
}
