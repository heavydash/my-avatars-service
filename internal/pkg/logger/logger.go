package logger

import (
	"context"
	"github.com/heavydash/my-avatars-service/internal/config"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"os"
)

// Logger — основной интерфейс логирования
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	InfoCtx(ctx context.Context, msg string, args ...any)
	ErrorCtx(ctx context.Context, msg string, args ...any)
	WarnCtx(ctx context.Context, msg string, args ...any)

	Sync() error
}

// slogLogger — реализация на базе slog
type slogLogger struct {
	*slog.Logger
}

// NewLogger создаёт логгер в зависимости от окружения
func NewLogger(cfg *config.ObservabilityConfig) Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     parseLevel(cfg.LogLevel),
		AddSource: cfg.Environment == "development",
	}

	if cfg.Environment == "development" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return &slogLogger{
		Logger: slog.New(handler),
	}
}

func parseLevel(env string) slog.Level {
	switch env {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithTrace — добавляет trace_id и span_id из контекста
func (l *slogLogger) WithTrace(ctx context.Context) *slog.Logger {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		return l.Logger.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return l.Logger
}

func (l *slogLogger) InfoCtx(ctx context.Context, msg string, args ...any) {
	l.WithTrace(ctx).Info(msg, args...)
}

func (l *slogLogger) ErrorCtx(ctx context.Context, msg string, args ...any) {
	l.WithTrace(ctx).Error(msg, args...)
}

func (l *slogLogger) WarnCtx(ctx context.Context, msg string, args ...any) {
	l.WithTrace(ctx).Warn(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

func (l *slogLogger) Sync() error {
	return nil
}
