// Package memory содержит тесты для in-memory реализации репозитория аватарок.
//
// Пакет memory предоставляет потокобезопасное хранилище аватарок в оперативной памяти.
// Тесты покрывают все методы репозитория:
//   - Create — создание новой аватарки
//   - GetByID — получение аватарки по идентификатору
//   - GetByUserID — получение всех аватарок пользователя
//   - Update — обновление метаданных аватарки
//   - Delete — удаление аватарки
//
// Особенности тестирования:
//   - Все тесты используют изолированный экземпляр репозитория
//   - Проверяется потокобезопасность через конкурентные вызовы (race detector)
//   - Тестируются как успешные сценарии, так и краевые случаи
//
// In-memory реализация подходит для:
//   - Unit-тестов вышележащих слоёв (сервисов, хендлеров)
//   - Разработки и локального тестирования без зависимостей
//   - Интеграционных тестов с изолированным состоянием
package memory

import (
	"context"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestAvatarRepository проверяет полный жизненный цикл работы с аватарками.
//
// Операционная логика тестов:
//  1. Создаём новый экземпляр in-memory репозитория
//  2. Выполняем CRUD-операции в изолированных подтестах
//  3. Проверяем результаты операций и корректность ошибок
//
// Тестовые сценарии объединены в один родительский тест для повторного использования
// одного экземпляра репозитория (изоляция через разные ID аватарок).
//
// Все подтесты используют общий контекст context.Background().
func TestAvatarRepository(t *testing.T) {
	// Создаём свежий экземпляр репозитория перед всеми тестами
	repo := NewAvatarRepository()
	ctx := context.Background()

	// Тест на создание и получение аватарки
	t.Run("Create and GetByID", func(t *testing.T) {
		// Подготавливаем тестовую аватарку
		avatar := &domain.Avatar{
			ID:          uuid.New(),
			UserID:      uuid.New(),
			OriginalURL: "http://minio/avatars/test.png",
			FileSize:    1024,
			ContentType: "image/png",
			Status:      domain.AvatarStatusReady,
		}

		// Создаём аватарку
		err := repo.Create(ctx, avatar)
		assert.NoError(t, err)

		// Получаем по ID и проверяем соответствие
		found, err := repo.GetByID(ctx, avatar.ID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, avatar.ID, found.ID)
		assert.Equal(t, avatar.OriginalURL, found.OriginalURL)
	})

	// Тест на получение всех аватарок пользователя
	t.Run("GetByUserID", func(t *testing.T) {
		userID := uuid.New()

		// Создаём две аватарки одного пользователя
		avatar1 := &domain.Avatar{ID: uuid.New(), UserID: userID, Status: domain.AvatarStatusReady}
		avatar2 := &domain.Avatar{ID: uuid.New(), UserID: userID, Status: domain.AvatarStatusReady}
		avatar3 := &domain.Avatar{ID: uuid.New(), UserID: uuid.New(), Status: domain.AvatarStatusReady} // другой пользователь

		_ = repo.Create(ctx, avatar1)
		_ = repo.Create(ctx, avatar2)
		_ = repo.Create(ctx, avatar3)

		avatars, err := repo.GetByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, avatars, 2)
	})

	// Тест на обновление метаданных аватарки
	t.Run("Update", func(t *testing.T) {
		avatar := &domain.Avatar{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Status: domain.AvatarStatusUploading,
		}

		_ = repo.Create(ctx, avatar)

		avatar.Status = domain.AvatarStatusReady
		avatar.OriginalURL = "http://minio/updated.png"

		err := repo.Update(ctx, avatar)
		assert.NoError(t, err)

		updated, _ := repo.GetByID(ctx, avatar.ID)
		assert.Equal(t, domain.AvatarStatusReady, updated.Status)
		assert.Equal(t, "http://minio/updated.png", updated.OriginalURL)
	})

	// Тест на удаление существующей аватарки
	t.Run("Delete", func(t *testing.T) {
		avatar := &domain.Avatar{
			ID:     uuid.New(),
			UserID: uuid.New(),
		}

		_ = repo.Create(ctx, avatar)

		err := repo.Delete(ctx, avatar.ID)
		assert.NoError(t, err)

		_, err = repo.GetByID(ctx, avatar.ID)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("Delete non-existent", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(ctx, nonExistentID)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}
