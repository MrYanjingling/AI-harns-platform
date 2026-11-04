package files

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/oklog/ulid/v2"
)

// Upload errors.
var (
	ErrFileTooLarge    = errors.New("file too large")
	ErrFileTypeInvalid = errors.New("file type invalid")
)

// Service uploads files to MinIO.
type Service struct {
	client   *minio.Client
	bucket   string
	maxBytes int64
}

// NewService builds a file upload service.
func NewService(client *minio.Client, bucket string, cfg Config) *Service {
	if cfg.MaxImageBytes <= 0 {
		cfg.MaxImageBytes = DefaultConfig().MaxImageBytes
	}
	return &Service{client: client, bucket: bucket, maxBytes: cfg.MaxImageBytes}
}

// UploadImage stores an image and returns its MinIO object key.
func (s *Service) UploadImage(ctx context.Context, header *multipart.FileHeader) (string, error) {
	if header == nil {
		return "", ErrFileTypeInvalid
	}
	if header.Size > s.maxBytes {
		return "", ErrFileTooLarge
	}

	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, s.maxBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > s.maxBytes {
		return "", ErrFileTooLarge
	}

	contentType := http.DetectContentType(data)
	ext, ok := imageExtension(contentType)
	if !ok {
		return "", ErrFileTypeInvalid
	}

	if err := s.ensureBucket(ctx); err != nil {
		return "", err
	}

	now := time.Now()
	key := fmt.Sprintf("images/%s/%s/%s.%s",
		now.Format("2006"), now.Format("01"), strings.ToLower(ulid.Make().String()), ext)
	_, err = s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *Service) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func imageExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return "jpg", true
	case "image/png":
		return "png", true
	case "image/webp":
		return "webp", true
	case "image/gif":
		return "gif", true
	default:
		return "", false
	}
}
