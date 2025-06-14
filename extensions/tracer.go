package extensions

import (
	"context"
	"log"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer is a struct that holds the OpenTelemetry tracer and its configuration.
type Tracer struct {
	ServiceName  string
	Version      string
	CollectorURL string
	tracer       oteltrace.Tracer
}

func NewTracer(serviceName, collectorURL string) *Tracer {
	return &Tracer{
		ServiceName:  serviceName,
		CollectorURL: collectorURL,
		tracer:       otel.Tracer(serviceName),
	}
}

func (t *Tracer) Tracer() oteltrace.Tracer {
	return t.tracer
}

// Init initializes the OpenTelemetry tracer.
func (t *Tracer) Init() func(context.Context) error {
	exporter, err := otlptrace.New(context.Background(), otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(t.CollectorURL),
	))

	if err != nil {
		log.Fatalln("Failed to create OTLP exporter:", err)
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", t.ServiceName),
			attribute.String("service.version", t.Version),
		),
	)

	if err != nil {
		log.Fatalln("Failed to create resource:", err)
	}

	otel.SetTracerProvider(
		trace.NewTracerProvider(
			trace.WithSampler(trace.AlwaysSample()),
			trace.WithBatcher(exporter),
			trace.WithResource(resources),
		),
	)

	return exporter.Shutdown
}

// Middleware is an HTTP middleware that traces requests using OpenTelemetry.
func (t *Tracer) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := t.Tracer().Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.remote_addr", r.RemoteAddr),
		)

		next(w, r.WithContext(ctx))
	}
}
