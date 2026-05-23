package minio

import (
	"context"
	"fmt"
	"github.com/heavydash/my-avatars-service/internal/config"
	"github.com/heavydash/my-avatars-service/internal/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"mime/multipart"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOStorage(cfg *config.Config) (storage.Storage, error) {
	client, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Создаём bucket
	err = client.MakeBucket(context.Background(), cfg.MinIO.Bucket, minio.MakeBucketOptions{})
	if err != nil {
		// Проверяем, существует ли уже
		exists, errBucketExists := client.BucketExists(context.Background(), cfg.MinIO.Bucket)
		if errBucketExists != nil || !exists {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: cfg.MinIO.Bucket,
	}, nil
}

func (s *MinIOStorage) Save(ctx context.Context, objectName string, file multipart.File, header *multipart.FileHeader) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, file, header.Size, minio.PutObjectOptions{
		ContentType: header.Header.Get("Content-Type"),
	})
	if err != nil {
		return "", err
	}

	return s.GetURL(objectName), nil
}

func (s *MinIOStorage) GetURL(objectName string) string {
	return fmt.Sprintf("http://localhost:9000/%s/%s", s.bucket, objectName)
}

func (s *MinIOStorage) Delete(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}
