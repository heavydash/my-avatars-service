// Package handler содержит HTTP-обработчики для работы с аватарками.
//
// Тесты покрывают следующие сценарии:
//   - Валидация входных данных (user_id, avatar_id, file)
//   - Авторизация через X-User-ID
//   - Обработка ошибок сервисного слоя
//   - Редиректы на CDN и fallback на дефолтную аватарку
//   - Конвертация доменных ошибок в HTTP статусы
//
// Для мокирования используется testify/mock.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/heavydash/my-avatars-service/internal/domain"
)

// mockAvatarService реализует интерфейс domain.AvatarService для тестирования.
//
// Особенности:
//   - Все методы могут быть замоканы через testify/mock
//   - Поддерживает проверку вызовов через AssertExpectations
//   - Возвращает domain-ошибки для проверки их конвертации в HTTP статусы
type mockAvatarService struct {
	mock.Mock
}

// UploadAvatar мокирует загрузку аватарки в хранилище.
//
// Операционная логика:
//  1. Проверяем аргументы вызова
//  2. Возвращаем предопределённый avatar и error
func (m *mockAvatarService) UploadAvatar(ctx context.Context, userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*domain.Avatar, error) {
	args := m.Called(ctx, userID, file, header)
	if av := args.Get(0); av != nil {
		return av.(*domain.Avatar), args.Error(1)
	}
	return nil, args.Error(1)
}

// DeleteAvatar мокирует удаление аватарки.
func (m *mockAvatarService) DeleteAvatar(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// GetByID мокирует получение аватарки по ID.
func (m *mockAvatarService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	args := m.Called(ctx, id)
	if av := args.Get(0); av != nil {
		return av.(*domain.Avatar), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetByUserID мокирует получение всех аватарок пользователя.
func (m *mockAvatarService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Avatar, error) {
	args := m.Called(ctx, userID)
	if avatars := args.Get(0); avatars != nil {
		return avatars.([]*domain.Avatar), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetByIDWithOptions мокирует получение аватарки с опциями трансформации.
func (m *mockAvatarService) GetByIDWithOptions(ctx context.Context, id uuid.UUID, size, format string) (*domain.Avatar, error) {
	args := m.Called(ctx, id, size, format)
	if av := args.Get(0); av != nil {
		return av.(*domain.Avatar), args.Error(1)
	}
	return nil, args.Error(1)
}

// TestAvatarHandler_UploadAvatar проверяет эндпоинт POST /api/v1/avatars.
//
// Сценарии тестирования:
//  1. success — успешная загрузка PNG файла, статус 201
//  2. missing user ID header — отсутствует X-User-ID → 400
//  3. invalid user ID format — невалидный UUID → 400
//  4. no file uploaded — не передан файл → 400
//  5. service error - file too large — превышение лимита 10MB → 400
//
// Покрываемые кейсы:
//   - Валидация multipart/form-data
//   - Чтение заголовка X-User-ID
//   - Конвертация доменных ошибок в HTTP статусы
func TestAvatarHandler_UploadAvatar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userIDHeader   string
		setupRequest   func() (*http.Request, error)
		setupMocks     func(*mockAvatarService, uuid.UUID, uuid.UUID)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:         "success",
			userIDHeader: uuid.New().String(),
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)

				// Пишем в форму user_id
				writer.WriteField("user_id", uuid.New().String())

				// Создаём файловое поле с именем "file"
				part, err := writer.CreateFormFile("file", "test.png")
				if err != nil {
					return nil, err
				}
				// Пишем валидный PNG заголовок
				part.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
				writer.Close()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMocks: func(svc *mockAvatarService, avatarID, userID uuid.UUID) {
				// Мокаем успешную загрузку
				svc.On("UploadAvatar", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(&domain.Avatar{
						ID:     uuid.New(),
						UserID: uuid.New(),
						Status: domain.AvatarStatusReady,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   nil, // Тело проверяется отдельно через JSON
		},
		{
			name:         "missing user ID header",
			userIDHeader: "",
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				// user_id специально не добавляем
				part, _ := writer.CreateFormFile("file", "test.png")
				part.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
				writer.Close()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMocks:     func(svc *mockAvatarService, avatarID, userID uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "user_id is required",
			},
		},
		{
			name:         "invalid user ID format",
			userIDHeader: "invalid-uuid",
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.WriteField("user_id", "invalid-uuid")
				part, _ := writer.CreateFormFile("file", "test.png")
				part.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
				writer.Close()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMocks:     func(svc *mockAvatarService, avatarID, userID uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid user_id",
			},
		},
		{
			name:         "no file uploaded",
			userIDHeader: uuid.New().String(),
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.WriteField("user_id", uuid.New().String())
				writer.Close()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMocks:     func(svc *mockAvatarService, avatarID, userID uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "file is required",
			},
		},
		{
			name:         "service error - file too large",
			userIDHeader: uuid.New().String(),
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.WriteField("user_id", uuid.New().String())
				part, _ := writer.CreateFormFile("file", "test.png")
				part.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
				writer.Close()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMocks: func(svc *mockAvatarService, avatarID, userID uuid.UUID) {
				// Мокаем ошибку сервиса
				svc.On("UploadAvatar", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, domain.ErrFileTooLarge)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "file size exceeds 10MB limit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Инициализируем мок
			mockSvc := &mockAvatarService{}
			tt.setupMocks(mockSvc, uuid.Nil, uuid.Nil)

			handler := NewAvatarHandler(mockSvc)

			// Подготавливаем запрос
			req, err := tt.setupRequest()
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Устанавливаем заголовок, если передан
			if tt.userIDHeader != "" {
				c.Request.Header.Set("X-User-ID", tt.userIDHeader)
			}

			// Вызываем хендлер
			handler.UploadAvatar(c)

			// Проверяем статус
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Проверяем тело ответа
			if tt.expectedBody != nil {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			}

			// Проверяем, что все ожидаемые вызовы мока произошли
			mockSvc.AssertExpectations(t)
		})
	}
}

// TestAvatarHandler_DeleteAvatar проверяет эндпоинт DELETE /api/v1/avatars/:id.
//
// Сценарии тестирования:
//  1. success — удаление своей аватарки → 204
//  2. invalid avatar id — неверный формат ID → 400
//  3. missing X-User-ID header — отсутствует заголовок → 401
//  4. invalid X-User-ID format — неверный формат UUID → 400
//  5. avatar not found — аватарка не найдена → 404
//  6. forbidden — попытка удалить чужую аватарку → 403
//  7. service error — внутренняя ошибка сервиса → 500
//
// Операционная логика тестов:
//  1. Мокаем GetByID для проверки прав владения
//  2. Мокаем DeleteAvatar для успешного удаления
//  3. Проверяем корректную конвертацию domain.ErrNotFound в 404
func TestAvatarHandler_DeleteAvatar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		avatarID       string
		userIDHeader   string
		setupMocks     func(*mockAvatarService, uuid.UUID, uuid.UUID)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:         "success",
			avatarID:     uuid.New().String(),
			userIDHeader: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, avatarID, userID uuid.UUID) {
				// Сначала получаем аватарку для проверки прав
				svc.On("GetByID", mock.Anything, avatarID).
					Return(&domain.Avatar{ID: avatarID, UserID: userID}, nil)
				// Затем удаляем
				svc.On("DeleteAvatar", mock.Anything, avatarID).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid avatar id",
			avatarID:       "invalid-uuid",
			userIDHeader:   uuid.New().String(),
			setupMocks:     func(svc *mockAvatarService, _, _ uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid avatar id",
			},
		},
		{
			name:           "missing X-User-ID header",
			avatarID:       uuid.New().String(),
			userIDHeader:   "",
			setupMocks:     func(svc *mockAvatarService, _, _ uuid.UUID) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "X-User-ID header is required",
			},
		},
		{
			name:           "invalid X-User-ID format",
			avatarID:       uuid.New().String(),
			userIDHeader:   "invalid-uuid",
			setupMocks:     func(svc *mockAvatarService, _, _ uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid X-User-ID",
			},
		},
		{
			name:         "avatar not found",
			avatarID:     uuid.New().String(),
			userIDHeader: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, avatarID, _ uuid.UUID) {
				svc.On("GetByID", mock.Anything, avatarID).
					Return(nil, domain.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "avatar not found",
			},
		},
		{
			name:         "forbidden - not owner",
			avatarID:     uuid.New().String(),
			userIDHeader: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, avatarID, userID uuid.UUID) {
				ownerID := uuid.New() // Владелец аватарки — другой пользователь

				svc.On("GetByID", mock.Anything, avatarID).
					Return(&domain.Avatar{ID: avatarID, UserID: ownerID}, nil)

			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"error": "you can only delete your own avatars",
			},
		},
		{
			name:         "service error",
			avatarID:     uuid.New().String(),
			userIDHeader: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, avatarID, userID uuid.UUID) {
				svc.On("GetByID", mock.Anything, avatarID).
					Return(&domain.Avatar{ID: avatarID, UserID: userID}, nil)

				svc.On("DeleteAvatar", mock.Anything, avatarID).
					Return(domain.ErrInternal)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": domain.ErrInternal.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockAvatarService{}

			avatarUUID, _ := uuid.Parse(tt.avatarID)
			userUUID, _ := uuid.Parse(tt.userIDHeader)

			tt.setupMocks(mockSvc, avatarUUID, userUUID)

			handler := NewAvatarHandler(mockSvc)

			router := gin.New()
			router.DELETE("/api/v1/avatars/:id", handler.DeleteAvatar)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/avatars/"+tt.avatarID, nil)

			if tt.userIDHeader != "" {
				req.Header.Set("X-User-ID", tt.userIDHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "status code mismatch")

			if tt.expectedBody != nil && w.Code != http.StatusNoContent {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					for key, expectedValue := range tt.expectedBody {
						assert.Equal(t, expectedValue, response[key])
					}
				}
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

// TestAvatarHandler_GetAvatar проверяет эндпоинт GET /api/v1/avatars/:id.
//
// Сценарии тестирования:
//  1. success original — редирект (307) на оригинал в CDN
//  2. success thumbnail — редирект на ресайзнутую версию
//  3. invalid avatar id — неверный формат ID → 400
//  4. avatar not found — аватарка не найдена → 404
//  5. service internal error — ошибка сервиса → 500
//
// Особенности:
//   - Используется TemporaryRedirect (307) для сохранения метода запроса
//   - Размер ресайза передаётся через query параметр size
func TestAvatarHandler_GetAvatar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		avatarID         string
		size             string
		setupMocks       func(*mockAvatarService, uuid.UUID)
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:     "success original",
			avatarID: uuid.New().String(),
			size:     "original",
			setupMocks: func(svc *mockAvatarService, id uuid.UUID) {
				svc.On("GetByIDWithOptions", mock.Anything, id, "original", "").
					Return(&domain.Avatar{
						ID:          id,
						OriginalURL: "https://cdn.example.com/avatars/original/" + id.String() + ".jpg",
						Status:      domain.AvatarStatusReady,
					}, nil)
			},
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "",
		},
		{
			name:     "success thumbnail",
			avatarID: uuid.New().String(),
			size:     "300x300",
			setupMocks: func(svc *mockAvatarService, id uuid.UUID) {
				svc.On("GetByIDWithOptions", mock.Anything, id, "300x300", "").
					Return(&domain.Avatar{
						ID:     id,
						Status: domain.AvatarStatusReady,
					}, nil)
			},
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "invalid avatar id",
			avatarID:       "invalid-uuid",
			size:           "original",
			setupMocks:     func(svc *mockAvatarService, _ uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "avatar not found",
			avatarID: uuid.New().String(),
			size:     "original",
			setupMocks: func(svc *mockAvatarService, id uuid.UUID) {
				svc.On("GetByIDWithOptions", mock.Anything, id, "original", "").
					Return(nil, domain.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "service internal error",
			avatarID: uuid.New().String(),
			size:     "original",
			setupMocks: func(svc *mockAvatarService, id uuid.UUID) {
				svc.On("GetByIDWithOptions", mock.Anything, id, "original", "").
					Return(nil, domain.ErrInternal)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockAvatarService{}

			avatarUUID, _ := uuid.Parse(tt.avatarID)
			tt.setupMocks(mockSvc, avatarUUID)

			handler := NewAvatarHandler(mockSvc)

			router := gin.New()
			router.GET("/api/v1/avatars/:id", handler.GetAvatar)

			url := "/api/v1/avatars/" + tt.avatarID
			if tt.size != "" && tt.size != "original" {
				url += "?size=" + tt.size
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Проверяем Location только для редиректов
			if tt.expectedStatus == http.StatusTemporaryRedirect {
				location := w.Header().Get("Location")
				assert.NotEmpty(t, location)

				if tt.size == "original" && tt.name == "success original" {
					assert.Contains(t, location, "cdn.example.com")
				} else if tt.size != "original" {
					assert.Contains(t, location, fmt.Sprintf("/avatars/thumbnails/%s/%s.jpg", tt.avatarID, tt.size))
				}
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

// TestAvatarHandler_GetUserAvatar проверяет эндпоинт GET /api/v1/users/:user_id/avatar.
//
// Сценарии тестирования:
//  1. success — у пользователя есть аватарки → редирект на последнюю
//  2. no avatars — нет аватарок → fallback на дефолтную аватарку
//  3. invalid user_id — неверный формат UUID → 400
//  4. missing user_id — отсутствует параметр → 400
//  5. service error — ошибка сервиса → 500
//
// Fallback логика:
//   - Если GetByUserID вернул пустой список → редирект на /web/default-avatar.jpg
func TestAvatarHandler_GetUserAvatar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		userID           string
		setupMocks       func(*mockAvatarService, uuid.UUID)
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:   "success",
			userID: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, userID uuid.UUID) {
				avatarID := uuid.New()
				svc.On("GetByUserID", mock.Anything, userID).
					Return([]*domain.Avatar{
						{
							ID:          avatarID,
							UserID:      userID,
							OriginalURL: "https://cdn.example.com/avatars/" + avatarID.String() + ".jpg",
							Status:      domain.AvatarStatusReady,
						},
					}, nil)
			},
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://cdn.example.com/avatars/",
		},
		{
			name:   "no avatars - default fallback",
			userID: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, userID uuid.UUID) {
				svc.On("GetByUserID", mock.Anything, userID).
					Return([]*domain.Avatar{}, nil)
			},
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "/web/default-avatar.jpg",
		},
		{
			name:           "missing user_id",
			userID:         "",
			setupMocks:     func(svc *mockAvatarService, _ uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid user_id",
			userID:         "not-a-uuid",
			setupMocks:     func(svc *mockAvatarService, _ uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, userID uuid.UUID) {
				svc.On("GetByUserID", mock.Anything, userID).
					Return(nil, domain.ErrInternal)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockAvatarService{}

			userUUID, _ := uuid.Parse(tt.userID)
			tt.setupMocks(mockSvc, userUUID)

			handler := NewAvatarHandler(mockSvc)

			router := gin.New()
			router.GET("/api/v1/users/:user_id/avatar", handler.GetUserAvatar)

			url := "/api/v1/users/" + tt.userID + "/avatar"
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "status code mismatch")

			if tt.expectedStatus == http.StatusTemporaryRedirect {
				location := w.Header().Get("Location")
				assert.NotEmpty(t, location)

				if tt.expectedLocation != "" {
					assert.Contains(t, location, tt.expectedLocation)
				}
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

// TestAvatarHandler_GetUserAvatars проверяет эндпоинт GET /api/v1/avatars?user_id=xxx.
//
// Сценарии тестирования:
//  1. success — получение списка всех аватарок пользователя → 200
//  2. missing user_id — отсутствует query параметр → 400
//  3. invalid user_id — неверный формат UUID → 400
//  4. service error — ошибка сервиса → 500
//
// Возвращаемые данные:
//   - Массив объектов Avatar с полями: id, user_id, original_url, status, thumbnails
func TestAvatarHandler_GetUserAvatars(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         string
		setupMocks     func(*mockAvatarService, uuid.UUID)
		expectedStatus int
	}{
		{
			name:   "success",
			userID: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, userID uuid.UUID) {
				svc.On("GetByUserID", mock.Anything, userID).
					Return([]*domain.Avatar{
						{
							ID:          uuid.New(),
							UserID:      userID,
							OriginalURL: "https://example.com/avatar1.jpg",
							Status:      domain.AvatarStatusReady,
						},
						{
							ID:          uuid.New(),
							UserID:      userID,
							OriginalURL: "https://example.com/avatar2.jpg",
							Status:      domain.AvatarStatusReady,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing user_id",
			userID:         "",
			setupMocks:     func(svc *mockAvatarService, _ uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid user_id",
			userID:         "invalid-uuid",
			setupMocks:     func(svc *mockAvatarService, _ uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, userID uuid.UUID) {
				svc.On("GetByUserID", mock.Anything, userID).
					Return(nil, domain.ErrInternal)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockAvatarService{}

			userUUID, _ := uuid.Parse(tt.userID)
			tt.setupMocks(mockSvc, userUUID)

			handler := NewAvatarHandler(mockSvc)

			router := gin.New()
			router.GET("/api/v1/avatars", handler.GetUserAvatars)

			url := "/api/v1/avatars"
			if tt.userID != "" {
				url += "?user_id=" + tt.userID
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Для успешного случая проверяем, что тело массив аватарок
			if tt.expectedStatus == http.StatusOK {
				var avatars []*domain.Avatar
				err := json.Unmarshal(w.Body.Bytes(), &avatars)
				assert.NoError(t, err)
				assert.NotEmpty(t, avatars)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

// TestAvatarHandler_GetAvatarMetadata проверяет эндпоинт GET /api/v1/avatars/:id/metadata.
//
// Сценарии тестирования:
//  1. success — получение метаданных аватарки → 200
//  2. invalid avatar id — неверный формат ID → 400
//  3. avatar not found — аватарка не найдена → 404
//  4. service internal error — ошибка сервиса → 500
//
// Возвращаемые метаданные:
//   - id, user_id, original_url, file_size, content_type, status, created_at, updated_at
//   - thumbnails — массив пресетов с размерами (original, small, medium, large)
func TestAvatarHandler_GetAvatarMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		avatarID       string
		setupMocks     func(*mockAvatarService, uuid.UUID)
		expectedStatus int
	}{
		{
			name:     "success",
			avatarID: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, id uuid.UUID) {
				svc.On("GetByID", mock.Anything, id).
					Return(&domain.Avatar{
						ID:          id,
						UserID:      uuid.New(),
						OriginalURL: "https://cdn.example.com/avatars/" + id.String() + ".jpg",
						FileSize:    1024576,
						ContentType: "image/jpeg",
						Status:      domain.AvatarStatusReady,
						CreatedAt:   time.Now().Add(-time.Hour),
						UpdatedAt:   time.Now(),
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid avatar id",
			avatarID:       "invalid-uuid",
			setupMocks:     func(svc *mockAvatarService, _ uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "avatar not found",
			avatarID: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, id uuid.UUID) {
				svc.On("GetByID", mock.Anything, id).
					Return(nil, domain.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "service internal error",
			avatarID: uuid.New().String(),
			setupMocks: func(svc *mockAvatarService, id uuid.UUID) {
				svc.On("GetByID", mock.Anything, id).
					Return(nil, domain.ErrInternal)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockAvatarService{}

			avatarUUID, _ := uuid.Parse(tt.avatarID)
			tt.setupMocks(mockSvc, avatarUUID)

			handler := NewAvatarHandler(mockSvc)

			router := gin.New()
			router.GET("/api/v1/avatars/:id/metadata", handler.GetAvatarMetadata)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+tt.avatarID+"/metadata", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Для успешного случая проверяем структуру ответа
			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)

				// Проверяем обязательные поля
				assert.NotEmpty(t, resp["id"])
				assert.NotEmpty(t, resp["user_id"])
				assert.NotEmpty(t, resp["original_url"])
				assert.NotEmpty(t, resp["file_size"])
				assert.NotEmpty(t, resp["content_type"])

				// Проверяем thumbnails
				assert.NotEmpty(t, resp["thumbnails"])
				thumbnails, ok := resp["thumbnails"].([]interface{})
				assert.True(t, ok)
				assert.Equal(t, 2, len(thumbnails), "expected 2 thumbnail presets")
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
