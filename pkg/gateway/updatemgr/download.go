package updatemgr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type downloader struct {
	client            *http.Client
	allowedHosts      []string
	allowInsecureHTTP bool
}

func newDownloader() *downloader {
	client := &http.Client{
		Timeout: 30 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many download redirects")
			}
			if _, err := validateDownloadURL(req.URL.String(), nil, true); err != nil {
				return err
			}
			return nil
		},
	}
	return &downloader{
		client:            client,
		allowInsecureHTTP: true,
	}
}

func (d *downloader) fetch(ctx context.Context, rawURL string, maxBytes int64) ([]byte, error) {
	if _, err := validateDownloadURL(rawURL, d.allowedHosts, d.allowInsecureHTTP); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := d.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", rawURL, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status %d", rawURL, response.StatusCode)
	}
	limit := maxBytes
	if limit <= 0 {
		limit = defaultMaxManifest
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rawURL, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download %s exceeds %d bytes", rawURL, limit)
	}
	return data, nil
}

func (d *downloader) downloadTo(ctx context.Context, rawURL, target string, maxBytes int64) error {
	if _, err := validateDownloadURL(rawURL, d.allowedHosts, d.allowInsecureHTTP); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	response, err := d.client.Do(request)
	if err != nil {
		return fmt.Errorf("download %s: %w", rawURL, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %d", rawURL, response.StatusCode)
	}
	if maxBytes > 0 && response.ContentLength > maxBytes {
		return fmt.Errorf("download %s exceeds %d bytes", rawURL, maxBytes)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create package directory: %w", err)
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create package file: %w", err)
	}
	var written int64
	copyErr := error(nil)
	if maxBytes > 0 {
		written, copyErr = io.Copy(file, io.LimitReader(response.Body, maxBytes+1))
		if copyErr == nil && written > maxBytes {
			copyErr = fmt.Errorf("download %s exceeds %d bytes", rawURL, maxBytes)
		}
	} else {
		written, copyErr = io.Copy(file, response.Body)
	}
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(target)
		return fmt.Errorf("download %s: %w", rawURL, copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(target)
		return fmt.Errorf("close package file: %w", closeErr)
	}
	if written == 0 {
		_ = os.Remove(target)
		return fmt.Errorf("download %s: empty body", rawURL)
	}
	return nil
}
