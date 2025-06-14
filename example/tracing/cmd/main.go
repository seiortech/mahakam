package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/radenrishwan/mahakam"
	"github.com/radenrishwan/mahakam/extensions"
	"github.com/radenrishwan/mahakam/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	serviceName  = "example-tracing"
	collectorURL = "jaeger:4317"
)

func main() {
	tracer := extensions.NewTracer(serviceName, collectorURL)

	cleanup := tracer.Init()
	defer cleanup(context.Background())

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, span := tracer.Tracer().Start(r.Context(), "handle_root")
		defer span.End()

		span.SetAttributes(attribute.String("handler", "root"))
		w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Tracer().Start(r.Context(), "handle_foo")
		defer span.End()

		span.SetAttributes(attribute.String("handler", "foo"))

		fetchData(ctx, tracer.Tracer())
		queryDatabase(ctx, tracer.Tracer())

		w.Write([]byte("foo"))
	})

	mux.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Tracer().Start(r.Context(), "handle_bar")
		defer span.End()

		span.SetAttributes(attribute.String("handler", "bar"))

		fetchData(ctx, tracer.Tracer())

		w.Write([]byte("bar"))
	})

	s := mahakam.NewServer("0.0.0.0:8080", mux)
	s.Use(middleware.Logger)
	s.Use(tracer.Middleware)

	if err := s.ListenAndServe(); err != nil {
		log.Fatalln("Failed to start server:", err)
	}
}

func fetchData(ctx context.Context, tracer trace.Tracer) {
	_, span := tracer.Start(ctx, "fetch_data")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "fetch_data"),
		attribute.Int("duration_ms", 2000),
	)

	time.Sleep(2 * time.Second)
}

func queryDatabase(ctx context.Context, tracer trace.Tracer) {
	_, span := tracer.Start(ctx, "query_database")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "query_database"),
		attribute.String("db.type", "postgres"),
		attribute.Int("duration_ms", 1000),
	)

	time.Sleep(1 * time.Second)
}
