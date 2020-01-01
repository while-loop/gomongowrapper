# gomongowrapper
MongoDB Go wrapper source code

## Table of contents
- [End to end example](#end-to-end-example)
- [Traces](#traces)
- [Metrics](#metrics)

## End to end example
With a MongoDB server running at "localhost:27017" and running this example with Go

```go
package main

import (
	"context"
	"github.com/while-loop/gomongowrapper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporter/trace/stackdriver"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log"
	"time"
)

func main() {
	// Enabling the OpenTelemetry exporter.
	// Just using Stackdriver since it has both Tracing and Metrics
	// and is easy to whip up. Add your desired one here.
    sde, err := stackdriver.NewExporter(
        stackdriver.WithProjectID("census-demos"),
    )
    if err != nil {
        log.Fatalf("Failed to create Stackdriver exporter: %v", err)
    }
    tp, err := sdktrace.NewProvider(
        sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
        sdktrace.WithSyncer(sde),
    )
    global.SetTraceProvider(tp)

	defer func() {
		<-time.After(2 * time.Minute)
	}()

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
```

## Traces
![](/images/gomongowrapper-traces.png)

