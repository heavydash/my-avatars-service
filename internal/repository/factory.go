package repository

import (
	"fmt"
	"github.com/heavydash/my-avatars-service/internal/config"
	"github.com/heavydash/my-avatars-service/internal/repository/memory"
	"github.com/heavydash/my-avatars-service/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewAvatarRepository возвращает нужный репозиторий
// с fallback-логикой в зависимости от окружения
func NewAvatarRepository(cfg *config.Config, db *pgxpool.Pool) (AvatarRepository, error) {
	switch cfg.Server.Env {
	case "test":
		fmt.Println("Using In-Memory Avatar Repository (test mode)")
		return memory.NewAvatarRepository(), nil

	case "development":
		// Можно переключать через ENV, но по умолчанию in-memory
		fmt.Println("Using In-Memory Avatar Repository (development mode)")
		return memory.NewAvatarRepository(), nil

	default:
		// production, staging — только Postgres
		if db == nil {
			return nil, fmt.Errorf("postgres connection pool is required in %s environment", cfg.Server.Env)
		}
		fmt.Println("Using Postgres Avatar Repository")
		return postgres.NewAvatarRepository(db), nil
	}
}
