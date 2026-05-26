package logger

import (
	"github.com/heavydash/my-avatars-service/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger — основной интерфейс логирования
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	With(fields ...zap.Field) Logger
	Sync() error
}

// ZapLogger — реализация на базе zap
type ZapLogger struct {
	logger *zap.Logger
}

// New создаёт логгер в зависимости от окружения
func New(cfg *config.Config) (Logger, error) {
	var zapCfg zap.Config

	if cfg.Server.Debug {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	zapLogger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	return &ZapLogger{logger: zapLogger}, nil
}

func (l *ZapLogger) Debug(msg string, fields ...zap.Field) { l.logger.Debug(msg, fields...) }
func (l *ZapLogger) Info(msg string, fields ...zap.Field)  { l.logger.Info(msg, fields...) }
func (l *ZapLogger) Warn(msg string, fields ...zap.Field)  { l.logger.Warn(msg, fields...) }
func (l *ZapLogger) Error(msg string, fields ...zap.Field) { l.logger.Error(msg, fields...) }
func (l *ZapLogger) Fatal(msg string, fields ...zap.Field) { l.logger.Fatal(msg, fields...) }

func (l *ZapLogger) With(fields ...zap.Field) Logger {
	return &ZapLogger{logger: l.logger.With(fields...)}
}

func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}
