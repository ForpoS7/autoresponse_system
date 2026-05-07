package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client *minio.Client
	bucket string
}

type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

func NewMinIOClient(endpoint, accessKey, secretKey, bucket string) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("minio init: %w", err)
	}

	c := &MinIOClient{client: client, bucket: bucket}
	if err := c.ensureBucket(context.Background()); err != nil {
		return nil, err
	}

	log.Printf("MinIO connected: endpoint=%s bucket=%s", endpoint, bucket)
	return c, nil
}

func (c *MinIOClient) ensureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("bucket check: %w", err)
	}
	if !exists {
		if err := c.client.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("bucket create: %w", err)
		}
		log.Printf("MinIO bucket created: %s", c.bucket)
	}
	return nil
}

// Upload загружает JSON-данные в cold storage.
// Путь: cold/{prefix}/YYYY/MM/DD/{filename}.json
func (c *MinIOClient) Upload(ctx context.Context, key string, data []byte) error {
	_, err := c.client.PutObject(ctx, c.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "application/json"},
	)
	if err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	log.Printf("MinIO uploaded: %s (%d bytes)", key, len(data))
	return nil
}

// Download скачивает объект из cold storage.
func (c *MinIOClient) Download(ctx context.Context, key string) ([]byte, error) {
	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", key, err)
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

// List перечисляет объекты в cold storage с заданным префиксом.
func (c *MinIOClient) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var result []ObjectInfo
	for obj := range c.client.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		result = append(result, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
		})
	}
	return result, nil
}
