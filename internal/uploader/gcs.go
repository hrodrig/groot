package uploader

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
)

type gcsUploader struct {
	cfg     config.GCSUploadCfg
	timeout time.Duration
}

func newGCSUploader(cfg config.GCSUploadCfg, timeout time.Duration) Uploader {
	return &gcsUploader{cfg: cfg, timeout: uploadTimeout(timeout)}
}

func (u *gcsUploader) Provider() string { return "gcs" }

func (u *gcsUploader) Upload(ctx context.Context, archivePath string, summary collector.Summary) (*Result, error) {
	bucket := strings.TrimSpace(u.cfg.Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	key := objectKey(u.cfg.KeyPrefix, archivePath)
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	client, err := storage.NewClient(ctx, gcsClientOptions()...)
	if err != nil {
		return nil, fmt.Errorf("gcs client: %w", err)
	}
	defer client.Close()

	w := client.Bucket(bucket).Object(key).NewWriter(ctx)
	w.ContentType = firstNonEmpty(strings.TrimSpace(u.cfg.ContentType), "application/gzip")
	if cc := strings.TrimSpace(u.cfg.CacheControl); cc != "" {
		w.CacheControl = cc
	}
	if kms := strings.TrimSpace(u.cfg.KMSKey); kms != "" {
		w.KMSKeyName = kms
	}
	if acl := strings.TrimSpace(u.cfg.PredefinedACL); acl != "" {
		w.PredefinedACL = acl
	}
	w.Metadata = mergeMetadata(u.cfg.Metadata, summary.RunID)

	if _, err := io.Copy(w, f); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("write object: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	uri := fmt.Sprintf("gs://%s/%s", bucket, key)
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat archive: %w", err)
	}
	return &Result{Provider: "gcs", URI: uri, Key: key, Size: info.Size()}, nil
}

func gcsClientOptions() []option.ClientOption {
	if ep := os.Getenv("STORAGE_EMULATOR_HOST"); ep != "" {
		return []option.ClientOption{option.WithEndpoint("http://" + strings.TrimPrefix(ep, "http://"))}
	}
	return nil
}
