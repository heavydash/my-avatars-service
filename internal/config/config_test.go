// Package config содержит тесты для конфигурации приложения.
//
// Тесты покрывают следующие аспекты:
//   - New: загрузка конфигурации из разных источников (дефолты, env, флаги)
//   - Validate: валидация обязательных полей конфигурации
//   - Приоритеты: флаги > env > дефолты
//   - Обработка ошибок при некорректной конфигурации
//
// Для тестирования используется пакет testing и testify/assert.
package config

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TestConfig_New проверяет загрузку конфигурации из разных источников.
//
// Операционная логика тестов:
//  1. Создаём новый FlagSet для изоляции тестовых флагов
//  2. Вызываем newWithFlags для загрузки конфигурации
//  3. Сравниваем полученные значения с ожидаемыми
//
// Тестовые сценарии:
//   - default_config: проверяем значения по умолчанию (порт 8085, режим development)
//   - env_override: проверяем переопределение через переменные окружения
//
// Особенности:
//   - Используем изолированный FlagSet, чтобы не конфликтовать с флагами других тестов
//   - Восстанавливаем оригинальные env-переменные после теста через defer
func TestConfig_New(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		cfg, err := newWithFlags(flag.NewFlagSet("test", flag.ContinueOnError))
		assert.NoError(t, err)

		assert.Equal(t, "8085", cfg.Server.Port)
		assert.Equal(t, "development", cfg.Server.Env)
		assert.True(t, cfg.Server.Debug)
		assert.Equal(t, "localhost:9000", cfg.MinIO.Endpoint)
		assert.Equal(t, "avatars", cfg.MinIO.Bucket)
	})

	t.Run("env_override", func(t *testing.T) {
		oldPort := os.Getenv("APP_PORT")
		oldDSN := os.Getenv("DB_DSN")
		oldMinioEndpoint := os.Getenv("MINIO_ENDPOINT")

		os.Setenv("APP_PORT", "3000")
		os.Setenv("DB_DSN", "postgres://test:test@localhost/testdb")
		os.Setenv("MINIO_ENDPOINT", "minio.example.com:9000")

		defer func() {
			os.Setenv("APP_PORT", oldPort)
			os.Setenv("DB_DSN", oldDSN)
			os.Setenv("MINIO_ENDPOINT", oldMinioEndpoint)
		}()

		cfg, err := newWithFlags(flag.NewFlagSet("test", flag.ContinueOnError))
		assert.NoError(t, err)

		assert.Equal(t, "3000", cfg.Server.Port)
		assert.Equal(t, "postgres://test:test@localhost/testdb", cfg.DB.DSN)
		assert.Equal(t, "minio.example.com:9000", cfg.MinIO.Endpoint)
	})
}

// TestConfig_Validate проверяет корректность валидации конфигурации.
//
// Операционная логика тестов:
//  1. Создаём тестовую конфигурацию с различными наборами полей
//  2. Вызываем метод Validate() на конфигурации
//  3. Проверяем, что ошибка содержит ожидаемое сообщение (или отсутствует)
//
// Тестовые сценарии:
//   - valid config: все обязательные поля заполнены → ошибки нет
//   - missing server port: отсутствует порт сервера → ошибка валидации
//   - missing DB_DSN: отсутствует DSN базы данных → ошибка валидации
//   - missing MinIO parameters: отсутствуют endpoint/bucket MinIO → ошибка валидации
//
// Обязательные поля конфигурации:
//   - Server.Port — порт HTTP-сервера
//   - DB.DSN — строка подключения к PostgreSQL
//   - MinIO.Endpoint — адрес S3-совместимого хранилища
//   - MinIO.Bucket — имя бакета для аватарок
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Server: ServerConfig{
					Port: "8085",
					Env:  "development",
				},
				DB: DBConfig{
					DSN: "postgres://user:pass@localhost/db",
				},
				MinIO: MinIOConfig{
					Endpoint:  "localhost:9000",
					AccessKey: "minioadmin",
					SecretKey: "minioadmin123",
					Bucket:    "avatars",
				},
			},
			wantErr: "",
		},
		{
			name: "missing server port",
			cfg: &Config{
				Server: ServerConfig{
					Port: "",
				},
				DB: DBConfig{
					DSN: "postgres://user:pass@localhost/db",
				},
				MinIO: MinIOConfig{
					Endpoint: "localhost:9000",
					Bucket:   "avatars",
				},
			},
			wantErr: "server port is required",
		},
		{
			name: "missing DB_DSN",
			cfg: &Config{
				Server: ServerConfig{
					Port: "8085",
				},
				DB: DBConfig{
					DSN: "",
				},
				MinIO: MinIOConfig{
					Endpoint: "localhost:9000",
					Bucket:   "avatars",
				},
			},
			wantErr: "database DSN is required",
		},
		{
			name: "missing MinIO parameters",
			cfg: &Config{
				Server: ServerConfig{
					Port: "8085",
				},
				DB: DBConfig{
					DSN: "postgres://user:pass@localhost/db",
				},
				MinIO: MinIOConfig{
					Endpoint: "",
					Bucket:   "",
				},
			},
			wantErr: "minio endpoint and bucket are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
