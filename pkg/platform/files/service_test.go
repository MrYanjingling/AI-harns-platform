package files

import (
	"bytes"
	"context"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageExtension(t *testing.T) {
	cases := map[string]string{
		"image/jpeg": "jpg",
		"image/png":  "png",
		"image/webp": "webp",
		"image/gif":  "gif",
	}
	for contentType, ext := range cases {
		got, ok := imageExtension(contentType)
		assert.True(t, ok, contentType)
		assert.Equal(t, ext, got)
	}
	_, ok := imageExtension("text/plain")
	assert.False(t, ok)
}

func multipartHeader(t *testing.T, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.bin")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content)) + 1024)
	require.NoError(t, err)
	t.Cleanup(func() { _ = form.RemoveAll() })
	return form.File["file"][0]
}

func TestUploadImageTooLarge(t *testing.T) {
	svc := NewService(nil, "bucket", Config{MaxImageBytes: 8})
	_, err := svc.UploadImage(context.Background(), multipartHeader(t, []byte("0123456789")))
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestUploadImageInvalidType(t *testing.T) {
	svc := NewService(nil, "bucket", Config{MaxImageBytes: 1024})
	_, err := svc.UploadImage(context.Background(), multipartHeader(t, []byte("not an image")))
	assert.ErrorIs(t, err, ErrFileTypeInvalid)
}

func TestNewServiceDefaultLimit(t *testing.T) {
	svc := NewService(nil, "bucket", Config{})
	assert.Equal(t, int64(10<<20), svc.maxBytes)
}
