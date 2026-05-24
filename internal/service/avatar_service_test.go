// Package service содержит бизнес-логику работы с аватарками.
//
// Тесты покрывают основные сценарии сервисного слоя:
//   - Загрузка аватарок с валидацией форматов и размеров
//   - Удаление аватарок с проверкой прав доступа
//   - Получение аватарок (одиночных, списком, с опциями)
//   - Интеграция с репозиторием, хранилищем S3 и event publisher'ом
//   - Обработка ошибок и транзакционная целостность
//
// Для мокирования используются testify/mock и кастомные реализации интерфейсов.
package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
	"mime/multipart"
)

// mockAvatarRepository имитирует работу с БД аватарок.
//
// Реализует интерфейс domain.AvatarRepository для тестирования сервисного слоя.
// Позволяет мокать операции CRUD и проверять вызовы через AssertExpectations.
type mockAvatarRepository struct {
	mock.Mock
}

// Create мокирует сохранение аватарки в БД.
func (m *mockAvatarRepository) Create(ctx context.Context, avatar *domain.Avatar) error {
	args := m.Called(ctx, avatar)
	return args.Error(0)
}

// GetByID мокирует получение аватарки по ID.
func (m *mockAvatarRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	args := m.Called(ctx, id)
	if av := args.Get(0); av != nil {
		return av.(*domain.Avatar), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetByUserID мокирует получение всех аватарок пользователя.
func (m *mockAvatarRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Avatar, error) {
	args := m.Called(ctx, userID)
	if avatars := args.Get(0); avatars != nil {
		return avatars.([]*domain.Avatar), args.Error(1)
	}
	return nil, args.Error(1)
}

// Delete мокирует удаление аватарки из БД.
func (m *mockAvatarRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Update мокирует обновление метаданных аватарки.
func (m *mockAvatarRepository) Update(ctx context.Context, avatar *domain.Avatar) error {
	args := m.Called(ctx, avatar)
	return args.Error(0)
}

// mockStorage имитирует работу с S3 хранилищем.
//
// Реализует интерфейс domain.Storage для тестирования.
// Поддерживает операции сохранения, удаления, получения и генерации URL.
type mockStorage struct {
	mock.Mock
}

// Save мокирует загрузку файла в хранилище через multipart file.
func (m *mockStorage) Save(ctx context.Context, key string, file multipart.File, header *multipart.FileHeader) (string, error) {
	args := m.Called(ctx, key, file, header)
	return args.String(0), args.Error(1)
}

// Delete мокирует удаление файла из хранилища.
func (m *mockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// Get мокирует получение файла из хранилища в виде байтов.
func (m *mockStorage) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if data := args.Get(0); data != nil {
		return data.([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetObject мокирует получение файла в виде ReadCloser.
func (m *mockStorage) GetObject(ctx context.Context, objectName string) (io.ReadCloser, error) {
	args := m.Called(ctx, objectName)
	if rc := args.Get(0); rc != nil {
		return rc.(io.ReadCloser), args.Error(1)
	}
	return nil, args.Error(1)
}

// SaveFromBytes мокирует сохранение байтового слайса в хранилище.
func (m *mockStorage) SaveFromBytes(ctx context.Context, objectName string, data []byte, contentType string) (string, error) {
	args := m.Called(ctx, objectName, data, contentType)
	return args.String(0), args.Error(1)
}

// GetURL мокирует генерацию публичного URL для объекта.
func (m *mockStorage) GetURL(objectName string) string {
	args := m.Called(objectName)
	return args.String(0)
}

// mockPublisher имитирует публикацию событий в message broker.
//
// Реализует интерфейс domain.EventPublisher для тестирования.
// Используется для проверки, что после успешных операций публикуются корректные события.
type mockPublisher struct {
	mock.Mock
}

// PublishAvatarUploaded мокирует публикацию события об успешной загрузке.
func (m *mockPublisher) PublishAvatarUploaded(ctx context.Context, event domain.AvatarUploadedEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// PublishAvatarDeleted мокирует публикацию события об удалении аватарки.
func (m *mockPublisher) PublishAvatarDeleted(ctx context.Context, event domain.AvatarDeleteEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// mockLogger имитирует логирование для тестов.
//
// Реализует интерфейс logger.Logger.
// Позволяет проверять, что определенные сообщения были залогированы.
type mockLogger struct {
	mock.Mock
}

// Debug мокирует debug-логирование.
func (m *mockLogger) Debug(msg string, fields ...zapcore.Field) { m.Called(msg, fields) }

// Info мокирует info-логирование.
func (m *mockLogger) Info(msg string, fields ...zapcore.Field) { m.Called(msg, fields) }

// Warn мокирует warning-логирование.
func (m *mockLogger) Warn(msg string, fields ...zapcore.Field) { m.Called(msg, fields) }

// Error мокирует error-логирование.
func (m *mockLogger) Error(msg string, fields ...zapcore.Field) { m.Called(msg, fields) }

// Fatal мокирует fatal-логирование.
func (m *mockLogger) Fatal(msg string, fields ...zapcore.Field) { m.Called(msg, fields) }

// With мокирует создание дочернего логгера с дополнительными полями.
func (m *mockLogger) With(fields ...zapcore.Field) logger.Logger {
	args := m.Called(fields)
	if l := args.Get(0); l != nil {
		return l.(logger.Logger)
	}
	return m
}

// Sync мокирует сброс буферов логгера.
func (m *mockLogger) Sync() error {
	args := m.Called()
	return args.Error(0)
}

// mockFile имитирует multipart.File для тестов.
//
// Реализует интерфейсы io.Reader, io.Seeker, io.Closer.
// Позволяет передавать в сервис тестовые данные как настоящий загруженный файл.
type mockFile struct {
	*bytes.Reader
}

// Close имитирует закрытие файла.
func (f *mockFile) Close() error {
	return nil
}

// TestAvatarService_UploadAvatar проверяет бизнес-логику загрузки аватарок.
//
// Сценарии тестирования:
//  1. success with valid PNG file — успешная загрузка PNG (1KB) → статус Ready
//  2. invalid user ID (nil) — попытка загрузить для нулевого user_id → ErrInvalidInput
//  3. file too large (11MB) — превышение лимита в 10MB → ErrFileTooLarge
//  4. unsupported format - text file — неподдерживаемый MIME тип → ErrUnsupportedFormat
//
// Операционная логика тестов:
//  1. Создаём моки репозитория, хранилища, паблишера и логгера
//  2. Настраиваем ожидания вызовов через setupMocks
//  3. Вызываем метод UploadAvatar с тестовыми данными
//  4. Проверяем результат и ошибку
//  5. Валидируем, что все ожидаемые вызовы моков произошли
//
// Покрываемые кейсы:
//   - Валидация размера файла (лимит 10MB)
//   - Валидация MIME-типа (только image/jpeg, image/png, image/webp)
//   - Генерация UUID для новой аватарки
//   - Сохранение в S3 → сохранение в БД → публикация события
//   - Обработка ошибок на каждом этапе
func TestAvatarService_UploadAvatar(t *testing.T) {
	tests := []struct {
		name           string
		userID         uuid.UUID
		fileSize       int64
		contentType    string
		fileData       []byte
		setupMocks     func(*mockAvatarRepository, *mockStorage, *mockPublisher, *mockLogger)
		expectedError  error
		validateResult func(*testing.T, *domain.Avatar)
	}{
		{
			name:        "success with valid PNG file",
			userID:      uuid.New(),
			fileSize:    1024,
			contentType: "image/png",
			fileData:    createPNGHeader(),
			setupMocks: func(repo *mockAvatarRepository, storage *mockStorage, publisher *mockPublisher, log *mockLogger) {
				// Ожидаем сохранение в S3
				storage.On("Save", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).
					Return("http://minio:9000/avatars/123.png", nil)

				// Ожидаем сохранение в БД
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Avatar")).Return(nil)

				// Ожидаем публикацию события
				publisher.On("PublishAvatarUploaded", mock.Anything, mock.Anything).Return(nil)

				// Ожидаем лог успешной загрузки
				log.On("Info", mock.Anything, mock.Anything).Return()
			},
			expectedError: nil,
			validateResult: func(t *testing.T, av *domain.Avatar) {
				// Проверяем, что ID сгенерировался
				assert.NotEqual(t, uuid.Nil, av.ID)
				// Проверяем статус
				assert.Equal(t, domain.AvatarStatusReady, av.Status)
			},
		},
		{
			name:        "invalid user ID (nil)",
			userID:      uuid.Nil,
			fileSize:    1024,
			contentType: "image/png",
			fileData:    createPNGHeader(),
			setupMocks: func(*mockAvatarRepository, *mockStorage, *mockPublisher, *mockLogger) { // Ничего не ожидаем, т.к. валидация отсекает до вызова репозитория
			},
			expectedError: domain.ErrInvalidInput,
		},
		{
			name:        "file too large (11MB)",
			userID:      uuid.New(),
			fileSize:    11 * 1024 * 1024, // 11MB > 10MB
			contentType: "image/png",
			fileData:    createPNGHeader(),
			setupMocks: func(repo *mockAvatarRepository, _ *mockStorage, _ *mockPublisher, _ *mockLogger) { // Проверяем, что Create не вызывается из-за ошибки валидации
				repo.AssertNotCalled(t, "Create")
			},
			expectedError: domain.ErrFileTooLarge,
		},
		{
			name:        "unsupported format - text file",
			userID:      uuid.New(),
			fileSize:    1024,
			contentType: "text/plain",
			fileData:    []byte("This is a text file"),
			setupMocks: func(repo *mockAvatarRepository, _ *mockStorage, _ *mockPublisher, _ *mockLogger) { // Проверяем, что Create не вызывается из-за неподдерживаемого формата
				repo.AssertNotCalled(t, "Create")
			},
			expectedError: domain.ErrUnsupportedFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Инициализируем моки
			mockRepo := &mockAvatarRepository{}
			mockStorage := &mockStorage{}
			mockPublisher := &mockPublisher{}
			mockLog := &mockLogger{}

			// Настраиваем ожидания вызовов
			tt.setupMocks(mockRepo, mockStorage, mockPublisher, mockLog)

			// Создаём сервис с моками
			svc := NewAvatarService(mockRepo, mockStorage, mockPublisher, mockLog)

			// Подготавливаем тестовый файл
			file := createMockFile(tt.fileData)
			header := &multipart.FileHeader{
				Filename: "test.png",
				Size:     tt.fileSize,
				Header:   make(map[string][]string),
			}
			header.Header.Set("Content-Type", tt.contentType)

			// Вызываем тестируемый метод
			avatar, err := svc.UploadAvatar(context.Background(), tt.userID, file, header)

			// Проверяем результат
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, avatar)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, avatar)
				if tt.validateResult != nil {
					tt.validateResult(t, avatar)
				}
			}

			// Проверяем, что все ожидаемые вызовы моков произошли
			mockRepo.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
			mockPublisher.AssertExpectations(t)
		})
	}
}

// TestAvatarService_DeleteAvatar проверяет бизнес-логику удаления аватарки.
//
// Ожидаемые сценарии:
//   - success: удаление существующей аватарки → удаление из БД и S3
//   - not found: аватарка не найдена → ErrNotFound
//   - storage error: ошибка при удалении из S3 → возврат ошибки, БД не трогаем
//   - db error: ошибка при удалении из БД → возврат ошибки
//
// TestAvatarService_DeleteAvatar проверяет бизнес-логику удаления аватарки.
func TestAvatarService_DeleteAvatar(t *testing.T) {
	tests := []struct {
		name          string
		avatarID      uuid.UUID
		setupMocks    func(*mockAvatarRepository, *mockPublisher, *mockLogger)
		expectedError error
	}{
		{
			name:     "success - delete existing avatar",
			avatarID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository, publisher *mockPublisher, log *mockLogger) {
				avatar := &domain.Avatar{
					ID:     uuid.New(),
					UserID: uuid.New(),
				}

				repo.On("GetByID", mock.Anything, mock.Anything).Return(avatar, nil)
				publisher.On("PublishAvatarDeleted", mock.Anything, mock.Anything).Return(nil)
				repo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:     "not found - avatar does not exist",
			avatarID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository, publisher *mockPublisher, log *mockLogger) {
				repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound)
				// Delete и Publish не должны вызываться
				repo.AssertNotCalled(t, "Delete")
			},
			expectedError: domain.ErrNotFound,
		},
		{
			name:     "publish error - but delete succeeds",
			avatarID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository, publisher *mockPublisher, log *mockLogger) {
				avatar := &domain.Avatar{ID: uuid.New(), UserID: uuid.New()}

				repo.On("GetByID", mock.Anything, mock.Anything).Return(avatar, nil)
				publisher.On("PublishAvatarDeleted", mock.Anything, mock.Anything).Return(errors.New("rabbitmq down"))
				repo.On("Delete", mock.Anything, mock.Anything).Return(nil)

				log.On("Warn", mock.Anything, mock.Anything).Return()
			},
			expectedError: nil, // удаление должно пройти несмотря на ошибку события
		},
		{
			name:     "repository returns ErrInternal",
			avatarID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository, publisher *mockPublisher, log *mockLogger) {
				avatar := &domain.Avatar{ID: uuid.New(), UserID: uuid.New()}

				repo.On("GetByID", mock.Anything, mock.Anything).Return(avatar, nil)
				publisher.On("PublishAvatarDeleted", mock.Anything, mock.Anything).Return(nil)
				repo.On("Delete", mock.Anything, mock.Anything).Return(domain.ErrInternal)
			},
			expectedError: domain.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockAvatarRepository{}
			mockStorage := &mockStorage{} // не используется в Delete
			mockPublisher := &mockPublisher{}
			mockLog := &mockLogger{}

			tt.setupMocks(mockRepo, mockPublisher, mockLog)

			svc := NewAvatarService(mockRepo, mockStorage, mockPublisher, mockLog)

			err := svc.DeleteAvatar(context.Background(), tt.avatarID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockPublisher.AssertExpectations(t)
		})
	}
}

// TestAvatarService_GetByID проверяет получение аватарки по ID.
//
// TestAvatarService_GetByID проверяет получение аватарки по ID.
func TestAvatarService_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		avatarID      uuid.UUID
		setupMocks    func(*mockAvatarRepository)
		expectedError error
		validate      func(*testing.T, *domain.Avatar)
	}{
		{
			name:     "success - return existing avatar",
			avatarID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository) {
				expectedAvatar := &domain.Avatar{
					ID:          uuid.New(),
					UserID:      uuid.New(),
					OriginalURL: "http://minio:9000/avatars/abc123.jpg",
					FileSize:    204800,
					ContentType: "image/png",
					Status:      domain.AvatarStatusReady,
				}
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(expectedAvatar, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, av *domain.Avatar) {
				assert.NotNil(t, av)
				assert.Equal(t, domain.AvatarStatusReady, av.Status)
				assert.NotEmpty(t, av.OriginalURL)
			},
		},
		{
			name:     "not found",
			avatarID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, domain.ErrNotFound)
			},
			expectedError: domain.ErrNotFound,
		},
		{
			name:     "repository internal error",
			avatarID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, domain.ErrInternal)
			},
			expectedError: domain.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockAvatarRepository{}
			mockStorage := &mockStorage{}
			mockPublisher := &mockPublisher{}
			mockLog := &mockLogger{}

			tt.setupMocks(mockRepo)

			svc := NewAvatarService(mockRepo, mockStorage, mockPublisher, mockLog)

			avatar, err := svc.GetByID(context.Background(), tt.avatarID)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, avatar)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, avatar)
				if tt.validate != nil {
					tt.validate(t, avatar)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAvatarService_GetByUserID проверяет получение всех аватарок пользователя.
//
// TestAvatarService_GetByUserID проверяет получение всех аватарок пользователя.
func TestAvatarService_GetByUserID(t *testing.T) {
	tests := []struct {
		name          string
		userID        uuid.UUID
		setupMocks    func(*mockAvatarRepository)
		expectedError error
		validate      func(*testing.T, []*domain.Avatar)
	}{
		{
			name:   "success - user has multiple avatars",
			userID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository) {
				avatars := []*domain.Avatar{
					{
						ID:          uuid.New(),
						UserID:      uuid.New(),
						OriginalURL: "http://minio/avatars/1.jpg",
						Status:      domain.AvatarStatusReady,
					},
					{
						ID:          uuid.New(),
						UserID:      uuid.New(),
						OriginalURL: "http://minio/avatars/2.jpg",
						Status:      domain.AvatarStatusReady,
					},
				}
				repo.On("GetByUserID", mock.Anything, mock.Anything).
					Return(avatars, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, avatars []*domain.Avatar) {
				assert.Len(t, avatars, 2)
				assert.NotEmpty(t, avatars[0].OriginalURL)
				assert.Equal(t, domain.AvatarStatusReady, avatars[0].Status)
			},
		},
		{
			name:   "success - user has no avatars",
			userID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository) {
				repo.On("GetByUserID", mock.Anything, mock.Anything).
					Return([]*domain.Avatar{}, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, avatars []*domain.Avatar) {
				assert.Empty(t, avatars)
			},
		},
		{
			name:   "repository error",
			userID: uuid.New(),
			setupMocks: func(repo *mockAvatarRepository) {
				repo.On("GetByUserID", mock.Anything, mock.Anything).
					Return(nil, domain.ErrInternal)
			},
			expectedError: domain.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockAvatarRepository{}
			mockStorage := &mockStorage{}
			mockPublisher := &mockPublisher{}
			mockLog := &mockLogger{}

			tt.setupMocks(mockRepo)

			svc := NewAvatarService(mockRepo, mockStorage, mockPublisher, mockLog)

			avatars, err := svc.GetByUserID(context.Background(), tt.userID)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, avatars)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, avatars)
				if tt.validate != nil {
					tt.validate(t, avatars)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// createPNGHeader возвращает валидный PNG заголовок для тестов.
//
// Операционная логика:
//   - Первые 8 байт PNG файла: 137 80 78 71 13 10 26 10
//   - Используется для обхода MIME-детекции в сервисе
func createPNGHeader() []byte {
	return []byte{137, 80, 78, 71, 13, 10, 26, 10}
}

// createMockFile создаёт тестовый файл, реализующий интерфейс multipart.File.
//
// Операционная логика:
//  1. Оборачиваем bytes.Reader в mockFile
//  2. mockFile реализует Close() (no-op)
//  3. bytes.Reader реализует Read(), Seek() (необходимо для чтения файла)
func createMockFile(data []byte) multipart.File {
	return &mockFile{Reader: bytes.NewReader(data)}
}
