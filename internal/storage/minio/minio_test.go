package minio

import (
	"bytes"
	"context"
	"github.com/heavydash/my-avatars-service/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"testing"
)

type mockMinIOClient struct {
	mock.Mock
}

func (m *mockMinIOClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64,
	opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, reader, objectSize, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *mockMinIOClient) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Get(0).(*minio.Object), args.Error(1)
}

func (m *mockMinIOClient) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Error(0)
}

func (m *mockMinIOClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	args := m.Called(ctx, bucketName)
	return args.Bool(0), args.Error(1)
}

func (m *mockMinIOClient) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	args := m.Called(ctx, bucketName, opts)
	return args.Error(0)
}

func TestMinIOStorage(t *testing.T) {
	cfg := &config.Config{
		MinIO: config.MinIOConfig{
			Endpoint:  "localhost:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin123",
			Bucket:    "avatars",
		},
	}

	t.Run("Save success", func(t *testing.T) {
		mockClient := &mockMinIOClient{}
		s := &MinIOStorage{client: mockClient,
			bucket:  cfg.MinIO.Bucket,
			baseURL: "http://localhost:9000",
		}

		fileContent := []byte("fake image")
		file := &mockFile{Reader: bytes.NewReader(fileContent)}
		header := &multipart.FileHeader{
			Filename: "test.png",
			Size:     int64(len(fileContent)),
			Header:   map[string][]string{"Content-Type": {"image/png"}},
		}

		mockClient.On("PutObject", mock.Anything, "avatars", "test.png", mock.Anything, mock.Anything, mock.Anything).
			Return(minio.UploadInfo{}, nil)

		url, err := s.Save(context.Background(), "test.png", file, header)

		require.NoError(t, err)
		assert.Contains(t, url, "/avatars/test.png")
		mockClient.AssertExpectations(t)
	})

	t.Run("Delete success", func(t *testing.T) {
		mockClient := &mockMinIOClient{}
		s := &MinIOStorage{client: mockClient, bucket: "avatars"}

		mockClient.On("RemoveObject", mock.Anything, "avatars", "old.png", mock.Anything).
			Return(nil)

		err := s.Delete(context.Background(), "old.png")
		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("GetURL", func(t *testing.T) {
		s := &MinIOStorage{bucket: "avatars",
			baseURL: "http://localhost:9000"}
		url := s.GetURL("user123/avatar.jpg")
		assert.Equal(t, "http://localhost:9000/avatars/user123/avatar.jpg", url)
	})
}

// mockFile — заглушка для multipart.File
type mockFile struct {
	*bytes.Reader
}

func (m *mockFile) Close() error { return nil }
