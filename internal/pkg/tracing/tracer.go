package tracing

import (
	"context"
	"github.com/heavydash/my-avatars-service/internal/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer инициализирует OpenTelemetry tracer с экспортом в Jaeger
func InitTracer(cfg *config.ObservabilityConfig) error {
	endpoint := cfg.OTELExporter
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	// Создаём exporter в Jaeger
	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithInsecure(), // для локальной разработки
		otlptracegrpc.WithEndpoint("localhost:4317"),
	)
	if err != nil {
		return err
	}

	serviceName := cfg.OTELServiceName
	if serviceName == "" {
		serviceName = "gophprofile"
	}

	// Tracer Provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	return nil
}
