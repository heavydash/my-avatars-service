package postgres

import (
	"context"
	"fmt"
	"github.com/heavydash/my-avatars-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool — обёртка над pgxpool.Pool
type Pool struct {
	*pgxpool.Pool
}

// New создаёт пул подключений к PostgreSQL
func New(ctx context.Context, cfg *config.Config) (*Pool, error) {
	if cfg.DB.DSN == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}

	pgxCfg, err := pgxpool.ParseConfig(cfg.DB.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Настройки пула из конфигурации
	pgxCfg.MaxConns = int32(cfg.DB.MaxConns)
	pgxCfg.MinConns = int32(cfg.DB.MinConns)
	pgxCfg.MaxConnLifetime = cfg.DB.MaxConnLifetime
	pgxCfg.MaxConnIdleTime = cfg.DB.MaxConnIdleTime
	pgxCfg.HealthCheckPeriod = cfg.DB.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Проверка соединения с таймаутом
	pingCtx, cancel := context.WithTimeout(ctx, cfg.DB.PingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Pool{Pool: pool}, nil
}

// Close закрывает пул соединений
func (p *Pool) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
