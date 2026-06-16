package tracing

import (
	"context"
	"github.com/heavydash/my-avatars-service/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"log"
	"os"
	"strings"
	"time"
)

// InitTracer инициализирует OpenTelemetry tracer с экспортом в Jaeger
func InitTracer(cfg *config.ObservabilityConfig) (*sdktrace.TracerProvider, error) {

	endpoint := cfg.OTELExporter
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	// внутри Kubernetes используем сервисное имя
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		endpoint = "jaeger:4317"
	}

	if endpoint == "none" || endpoint == "" {
		log.Println("OpenTelemetry tracing disabled")
		return nil, nil
	}

	// Таймаут берём из конфига или разумный дефолт
	timeout := cfg.OTELTimeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Создаём exporter в Jaeger
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(), // для локальной разработки
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		return nil, nil
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
