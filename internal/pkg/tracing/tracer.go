package tracing

import (
	"context"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer инициализирует OpenTelemetry tracer с экспортом в Jaeger
func InitTracer(serviceName string) error {
	// Создаём exporter в Jaeger
	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithInsecure(), // для локальной разработки
		otlptracegrpc.WithEndpoint("localhost:4317"),
	)
	if err != nil {
		return err
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
