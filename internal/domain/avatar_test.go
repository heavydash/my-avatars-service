// internal/domain/avatar_test.go
//
// Пакет domain содержит тесты для бизнес-сущности Avatar.
//
// Тестируются:
//   - NewAvatar: создание новой аватарки
//   - MarkAsReady: перевод в статус ready
//   - MarkAsFailed: перевод в статус failed с записью ошибки
package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewAvatar тестирует создание новой аватарки.
//
// Сценарии:
//   - success: успешное создание с валидными параметрами
//   - zero userID: создание с нулевым userID (валидный сценарий, но может быть ошибкой бизнес-логики)
func TestNewAvatar(t *testing.T) {
	tests := []struct {
		name        string    // имя теста
		userID      uuid.UUID // ID пользователя
		originalURL string    // оригинальный URL
		fileSize    int64     // размер файла
		contentType string    // MIME тип
	}{
		{
			name:        "success with valid user ID",
			userID:      uuid.New(),
			originalURL: "",
			fileSize:    1024,
			contentType: "image/png",
		},
		{
			name:        "success with nil user ID",
			userID:      uuid.Nil,
			originalURL: "",
			fileSize:    512,
			contentType: "image/jpeg",
		},
		{
			name:        "success with large file",
			userID:      uuid.New(),
			originalURL: "",
			fileSize:    5 * 1024 * 1024, // 5MB
			contentType: "image/webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем время до создания для проверки
			beforeCreate := time.Now().UTC()

			// Создаём аватарку
			avatar := NewAvatar(tt.userID, tt.originalURL, tt.fileSize, tt.contentType)

			// Проверяем результаты
			assert.NotEqual(t, uuid.Nil, avatar.ID)
			assert.Equal(t, tt.userID, avatar.UserID)
			assert.Equal(t, tt.originalURL, avatar.OriginalURL)
			assert.Equal(t, tt.fileSize, avatar.FileSize)
			assert.Equal(t, tt.contentType, avatar.ContentType)
			assert.Equal(t, AvatarStatusUploading, avatar.Status)
			assert.Empty(t, avatar.Error)

			// Проверка временных меток
			assert.True(t, avatar.CreatedAt.After(beforeCreate) || avatar.CreatedAt.Equal(beforeCreate))
			assert.True(t, avatar.UpdatedAt.After(beforeCreate) || avatar.UpdatedAt.Equal(beforeCreate))
			assert.Equal(t, avatar.CreatedAt, avatar.UpdatedAt)
		})
	}
}

// TestAvatar_MarkAsReady тестирует перевод аватарки в статус ready.
//
// Сценарии:
//   - success: успешный перевод из любого статуса в ready
//   - already ready: повторный вызов MarkAsReady не должен изменять статус
func TestAvatar_MarkAsReady(t *testing.T) {
	tests := []struct {
		name           string       // имя теста
		initialStatus  AvatarStatus // начальный статус
		initialError   string       // начальная ошибка
		expectedStatus AvatarStatus // ожидаемый статус
		expectedError  string       // ожидаемая ошибка
	}{
		{
			name:           "from uploading to ready",
			initialStatus:  AvatarStatusUploading,
			initialError:   "",
			expectedStatus: AvatarStatusReady,
			expectedError:  "",
		},
		{
			name:           "from failed to ready",
			initialStatus:  AvatarStatusFailed,
			initialError:   "upload error: connection timeout",
			expectedStatus: AvatarStatusReady,
			expectedError:  "upload error: connection timeout",
		},
		{
			name:           "from ready to ready (already ready)",
			initialStatus:  AvatarStatusReady,
			initialError:   "",
			expectedStatus: AvatarStatusReady,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём аватарку с определённым статусом
			avatar := &Avatar{
				ID:          uuid.New(),
				UserID:      uuid.New(),
				OriginalURL: "http://example.com/avatar.png",
				FileSize:    1024,
				ContentType: "image/png",
				Status:      tt.initialStatus,
				Error:       tt.initialError,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			}

			// Сохраняем время до обновления
			beforeUpdate := time.Now().UTC()

			// Ждём небольшой задержки для проверки UpdatedAt
			time.Sleep(1 * time.Millisecond)

			// Вызываем тестируемый метод
			avatar.MarkAsReady()

			// Проверяем результаты
			assert.Equal(t, tt.expectedStatus, avatar.Status)
			assert.Equal(t, tt.expectedError, avatar.Error)
			assert.True(t, avatar.UpdatedAt.After(beforeUpdate))
		})
	}
}

// TestAvatar_MarkAsFailed тестирует перевод аватарки в статус failed.
//
// Сценарии:
//   - success: успешный перевод в статус failed с записью ошибки
//   - overwrite error: перезапись существующей ошибки
func TestAvatar_MarkAsFailed(t *testing.T) {
	tests := []struct {
		name           string       // имя теста
		initialStatus  AvatarStatus // начальный статус
		initialError   string       // начальная ошибка
		errorMessage   string       // сообщение об ошибке для метода
		expectedStatus AvatarStatus // ожидаемый статус
		expectedError  string       // ожидаемая ошибка
	}{
		{
			name:           "from uploading to failed",
			initialStatus:  AvatarStatusUploading,
			initialError:   "",
			errorMessage:   "failed to save to S3: connection refused",
			expectedStatus: AvatarStatusFailed,
			expectedError:  "failed to save to S3: connection refused",
		},
		{
			name:           "from ready to failed",
			initialStatus:  AvatarStatusReady,
			initialError:   "",
			errorMessage:   "processing error: invalid image format",
			expectedStatus: AvatarStatusFailed,
			expectedError:  "processing error: invalid image format",
		},
		{
			name:           "overwrite existing error",
			initialStatus:  AvatarStatusFailed,
			initialError:   "old error: something went wrong",
			errorMessage:   "new error: disk full",
			expectedStatus: AvatarStatusFailed,
			expectedError:  "new error: disk full",
		},
		{
			name:           "empty error message",
			initialStatus:  AvatarStatusUploading,
			initialError:   "",
			errorMessage:   "",
			expectedStatus: AvatarStatusFailed,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём аватарку
			avatar := &Avatar{
				ID:          uuid.New(),
				UserID:      uuid.New(),
				OriginalURL: "http://example.com/avatar.png",
				FileSize:    1024,
				ContentType: "image/png",
				Status:      tt.initialStatus,
				Error:       tt.initialError,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			}

			// Сохраняем время до обновления
			beforeUpdate := time.Now().UTC()
			time.Sleep(1 * time.Millisecond)

			// Вызываем тестируемый метод
			avatar.MarkAsFailed(tt.errorMessage)

			// Проверяем результаты
			assert.Equal(t, tt.expectedStatus, avatar.Status)
			assert.Equal(t, tt.expectedError, avatar.Error)
			assert.True(t, avatar.UpdatedAt.After(beforeUpdate))
		})
	}
}

// TestAvatar_ConcurrentUpdates тестирует конкурентное обновление статусов.
func TestAvatar_ConcurrentUpdates(t *testing.T) {
	avatar := NewAvatar(uuid.New(), "", 1024, "image/png")

	// Запускаем конкурентные обновления
	done := make(chan bool)

	go func() {
		avatar.MarkAsReady()
		done <- true
	}()

	go func() {
		avatar.MarkAsFailed("concurrent error")
		done <- true
	}()

	// Ждём завершения обоих горутин
	<-done
	<-done

	// Проверяем, что статус имеет одно из допустимых значений
	assert.Contains(t, []AvatarStatus{AvatarStatusReady, AvatarStatusFailed}, avatar.Status)
}
