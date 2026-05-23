// Package config отвечает за загрузку и валидацию конфигурации приложения.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
	"time"
)

// Config — корневая структура конфигурации
type Config struct {
	Server ServerConfig `json:"server"`
	DB     DBConfig     `json:"db"`
	MinIO  MinIOConfig  `json:"minio"`
}

type ServerConfig struct {
	Env             string        `json:"env"`
	Port            string        `json:"port"`
	Debug           bool          `json:"debug"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
}

type DBConfig struct {
	DSN               string        `json:"dsn"`
	MaxConns          int           `json:"max_conns"`
	MinConns          int           `json:"min_conns"`
	MaxConnLifetime   time.Duration `json:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `json:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `json:"health_check_period"`
	PingTimeout       time.Duration `json:"ping_timeout"`
	MigrationTimeout  time.Duration `json:"migration_timeout"`
}

type MinIOConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	UseSSL    bool   `json:"use_ssl"`
}

// New — основная функция загрузки конфигурации
func New() (*Config, error) {
	return newWithFlags(flag.CommandLine)
}

func newWithFlags(fs *flag.FlagSet) (*Config, error) {
	cfg := defaultConfig()

	configFile := fs.String("c", "", "path to config file (json)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Printf("Warning: flag parse error: %v", err)
	}

	// JSON-файл
	if *configFile != "" {
		if err := loadFromJSON(*configFile, cfg); err != nil {
			log.Printf("Warning: failed to load JSON config: %v", err)
		}
	}

	loadDotEnv()

	overwriteFromEnv(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	log.Println(" Configuration loaded successfully")
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            "8085",
			Env:             "development",
			Debug:           true,
			ShutdownTimeout: 10 * time.Second,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    30 * time.Second,
		},
		DB: DBConfig{
			DSN:               "host=localhost port=5432 user=postgres password=postgres dbname=gophprofile sslmode=disable",
			MaxConns:          25,
			MinConns:          5,
			MaxConnLifetime:   1 * time.Hour,
			MaxConnIdleTime:   30 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
			PingTimeout:       5 * time.Second,
			MigrationTimeout:  30 * time.Second,
		},
		MinIO: MinIOConfig{
			Endpoint:  "localhost:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin123",
			Bucket:    "avatars",
			UseSSL:    false,
		},
	}
}

func loadDotEnv() {
	if err := godotenv.Load(); err == nil {
		log.Println(".env file loaded")
	} else {
		log.Println(".env file not found, using system environment variables")
	}
}

func overwriteFromEnv(cfg *Config) {
	// Server
	if v := os.Getenv("APP_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.Server.Env = v
	}
	if v := os.Getenv("APP_DEBUG"); v != "" {
		cfg.Server.Debug = v == "true" || v == "1"
	}

	// Database
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.DB.DSN = v
	}

	// MinIO
	if v := os.Getenv("MINIO_ENDPOINT"); v != "" {
		cfg.MinIO.Endpoint = v
	}
	if v := os.Getenv("MINIO_ACCESS_KEY"); v != "" {
		cfg.MinIO.AccessKey = v
	}
	if v := os.Getenv("MINIO_SECRET_KEY"); v != "" {
		cfg.MinIO.SecretKey = v
	}
	if v := os.Getenv("MINIO_BUCKET"); v != "" {
		cfg.MinIO.Bucket = v
	}
	if v := os.Getenv("MINIO_USE_SSL"); v != "" {
		cfg.MinIO.UseSSL = v == "true" || v == "1"
	}
}

func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.DB.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}
	if c.MinIO.Endpoint == "" || c.MinIO.Bucket == "" {
		return fmt.Errorf("minio endpoint and bucket are required")
	}
	return nil
}

func (c *ServerConfig) Addr() string {
	if !strings.HasPrefix(c.Port, ":") {
		return ":" + c.Port
	}
	return c.Port
}

// loadFromJSON загружает конфигурацию из JSON-файла.
func loadFromJSON(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}
