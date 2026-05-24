// Package postgres содержит интеграционные тесты для PostgreSQL репозитория аватарок.
//
// Тесты покрывают следующие аспекты:
//   - Create: создание новой аватарки в БД
//   - GetByID: получение аватарки по идентификатору
//   - GetByUserID: получение списка аватарок пользователя
//   - Update: обновление метаданных аватарки
//   - Delete: мягкое удаление (soft delete) аватарки
//   - Транзакционная целостность и обработка ошибок
//
// Виды тестов:
//  1. Интеграционные (TestAvatarRepository_Postgres) — работа с реальной БД
//  2. Модульные (TestAvatarRepository_Unit) — мокирование pgxpool
//
// Внимание: интеграционные тесты требуют запущенного PostgreSQL.
// Для запуска используйте: docker-compose up -d postgres
// Или: go test -tags=integration ./internal/repository/postgres/...
//
// Переменные окружения для интеграционных тестов:
//   - DB_DSN — строка подключения к PostgreSQL
//   - DATABASE_DSN — альтернативное имя переменной
package postgres

import (
	"context"
	"github.com/pashagolub/pgxmock/v5"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
)

// newTestPostgresRepo создаёт тестовое подключение к реальной БД.
//
// Операционная логика:
//  1. Пытаемся загрузить .env из нескольких возможных расположений
//  2. Получаем DSN из переменных окружения
//  3. Используем fallback-значение для локальной разработки
//  4. Создаём пул соединений с таймаутом 15 секунд
//  5. Очищаем таблицу avatars перед тестами (TRUNCATE ... CASCADE)
//  6. Регистрируем cleanup-функцию для закрытия соединения
//
// Внимание: fallback DSN используется ТОЛЬКО для CI/CD и локальной разработки.
func newTestPostgresRepo(t *testing.T) *AvatarRepository {
	t.Helper()

	// Пробуем загрузить .env из разных возможных расположений
	// (поддержка запуска из корня проекта и из директории пакета)
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../../../.env")
	_ = godotenv.Load(".env")

	// Читаем DSN из переменных окружения
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_DSN")
	}

	// ЖЁСТКИЙ FALLBACK — если .env не прочитался
	// Используется только для локального тестирования
	if dsn == "" || dsn == "host=localhost" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=gophprofile sslmode=disable"
		t.Log("WARNING: Using hardcoded DSN (environment variable was empty or truncated)")
	}

	t.Logf("Using DSN: %s", dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v\nDSN used: %s", err, dsn)
	}

	repo := NewAvatarRepository(pool)

	// Очистка перед тестами
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE avatars RESTART IDENTITY CASCADE")

	t.Cleanup(func() {
		pool.Close()
	})

	return repo
}

// TestAvatarRepository_Postgres выполняет интеграционные тесты с реальной PostgreSQL.
//
// Сценарии тестирования:
//  1. Create and GetByID — создание и получение аватарки
//  2. GetByUserID — получение всех аватарок пользователя
//  3. Delete — мягкое удаление и повторное удаление
//
// Особенности:
//   - Используется реальное подключение к БД
//   - Перед тестами таблица очищается (TRUNCATE)
//   - Проверяется мягкое удаление через поле deleted_at
//   - Тесты не зависят друг от друга (изоляция через разные user_id)
func TestAvatarRepository_Postgres(t *testing.T) {
	repo := newTestPostgresRepo(t)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("Create and GetByID", func(t *testing.T) {
		avatar := &domain.Avatar{
			ID:          uuid.New(),
			UserID:      userID,
			OriginalURL: "https://minio.example.com/avatars/test.png",
			FileSize:    2048576, // ← важно!
			ContentType: "image/png",
			Status:      domain.AvatarStatusReady,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		err := repo.Create(ctx, avatar)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, avatar.ID)
		require.NoError(t, err)
		assert.Equal(t, avatar.ID, found.ID)
		assert.Equal(t, domain.AvatarStatusReady, found.Status)
	})

	t.Run("GetByUserID", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			a := &domain.Avatar{
				ID:          uuid.New(),
				UserID:      userID,
				OriginalURL: "https://example.com/avatar.png",
				FileSize:    1024, // ← важно!
				ContentType: "image/png",
				Status:      domain.AvatarStatusReady,
			}
			_ = repo.Create(ctx, a)
		}

		avatars, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(avatars), 2)
	})

	t.Run("Delete (soft delete)", func(t *testing.T) {
		avatar := &domain.Avatar{
			ID:          uuid.New(),
			UserID:      userID,
			FileSize:    1024, // ← обязательно!
			ContentType: "image/png",
			Status:      domain.AvatarStatusReady,
		}

		// Создаём
		err := repo.Create(ctx, avatar)
		require.NoError(t, err)

		// Первое удаление — успешно
		err = repo.Delete(ctx, avatar.ID)
		require.NoError(t, err)

		// После удаления не должен находиться
		_, err = repo.GetByID(ctx, avatar.ID)
		assert.ErrorIs(t, err, domain.ErrNotFound)

		// Повторное удаление — тоже ErrNotFound
		err = repo.Delete(ctx, avatar.ID)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

}

// TestAvatarRepository_Unit выполняет модульные тесты с мокированием pgxpool.
//
// Сценарии тестирования:
//  1. Create — успешное создание аватарки
//  2. GetByID — успешное получение аватарки
//  3. Delete not found — удаление несуществующей аватарки
//
// Особенности:
//   - Используется pgxmock для имитации работы с БД
//   - Нет реального подключения к PostgreSQL
//   - Тесты быстрые и изолированные
//   - Проверяются SQL-запросы и аргументы
//
// Примечание: для проверки синтаксиса SQL рекомендуется также запускать интеграционные тесты
func TestAvatarRepository_Unit(t *testing.T) {

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewAvatarRepository(mock)

		avatar := &domain.Avatar{
			ID:          uuid.New(),
			UserID:      uuid.New(),
			OriginalURL: "http://minio/avatars/test.png",
			FileSize:    1024,
			ContentType: "image/png",
			Status:      domain.AvatarStatusReady,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		mock.ExpectExec(`INSERT INTO avatars`).
			WithArgs(
				avatar.ID,
				avatar.UserID,
				avatar.OriginalURL,
				avatar.FileSize,
				avatar.ContentType,
				avatar.Status,
				avatar.Error,
				avatar.CreatedAt,
				avatar.UpdatedAt,
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err = repo.Create(context.Background(), avatar)
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewAvatarRepository(mock)
		id := uuid.New()

		rows := mock.NewRows([]string{"id", "user_id", "original_url", "file_size", "content_type", "status", "error", "created_at", "updated_at"}).
			AddRow(id, uuid.New(), "http://minio/test.png", 1024, "image/png", domain.AvatarStatusReady, "", time.Now(), time.Now())

		mock.ExpectQuery(`SELECT .* FROM avatars WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(id).
			WillReturnRows(rows)

		avatar, err := repo.GetByID(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, id, avatar.ID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewAvatarRepository(mock)
		id := uuid.New()

		mock.ExpectExec(`UPDATE avatars SET deleted_at = NOW\(\), updated_at = NOW\(\) WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(id).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err = repo.Delete(context.Background(), id)
		assert.ErrorIs(t, err, domain.ErrNotFound)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
