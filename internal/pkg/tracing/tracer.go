package tracing

import (
	"context"
	"fmt"
	"github.com/heavydash/my-avatars-service/internal/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"strings"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer инициализирует OpenTelemetry tracer с экспортом в Jaeger
func InitTracer(cfg *config.ObservabilityConfig) (*sdktrace.TracerProvider, error) {
	endpoint := cfg.OTELExporter
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	if strings.Contains(endpoint, "jaeger") || endpoint == "" {
		endpoint = "localhost:4317"
	}

	// Создаём exporter в Jaeger
	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithInsecure(), // для локальной разработки
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
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
	return tp, nil
}
