package minio

import (
	"context"
	"github.com/minio/minio-go/v7"
	"io"
)

// MinIOClient — интерфейс для удобного мокирования minio.Client
type MinIOClient interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64,
		opts minio.PutObjectOptions) (minio.UploadInfo, error)

	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)

	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error

	BucketExists(ctx context.Context, bucketName string) (bool, error)

	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
}
