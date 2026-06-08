package uploader

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/hrodrig/groot/pkg/collector"
	"github.com/hrodrig/groot/pkg/config"
)

// Result describes a successful object upload.
type Result struct {
	Provider string
	URI      string
	Key      string
	ETag     string
	Size     int64
}

// Uploader pushes a collect archive to remote object storage.
type Uploader interface {
	Provider() string
	Upload(ctx context.Context, archivePath string, summary collector.Summary) (*Result, error)
}

// FanOut runs all enabled upload providers.
type FanOut struct {
	uploaders       []Uploader
	continueOnError bool
}

// NewFanOut builds uploaders from configuration.
func NewFanOut(cfg config.Config) *FanOut {
	u := cfg.Upload
	out := make([]Uploader, 0, 2)
	if u.S3.Enabled {
		out = append(out, newS3Uploader(u.S3, u.Timeout))
	}
	if u.GCS.Enabled {
		out = append(out, newGCSUploader(u.GCS, u.Timeout))
	}
	return &FanOut{uploaders: out, continueOnError: u.ContinueOnError}
}

// ShouldUpload reports whether upload is configured and not skipped by flags.
func ShouldUpload(cfg config.Config) bool {
	if !cfg.Upload.Enabled {
		return false
	}
	return cfg.Upload.S3.Enabled || cfg.Upload.GCS.Enabled
}

// Outcome is one provider upload attempt.
type Outcome struct {
	Result   *Result
	Err      error
	Provider string
}

// Upload pushes the archive to every enabled provider.
func (f *FanOut) Upload(ctx context.Context, archivePath string, summary collector.Summary) []Outcome {
	if len(f.uploaders) == 0 {
		return nil
	}
	var out []Outcome
	for _, up := range f.uploaders {
		res, err := up.Upload(ctx, archivePath, summary)
		out = append(out, Outcome{Result: res, Err: err, Provider: up.Provider()})
		if err != nil && !f.continueOnError {
			break
		}
	}
	return out
}

func objectKey(prefix, archivePath string) string {
	base := filepath.Base(archivePath)
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return base
	}
	return prefix + "/" + base
}

func uploadTimeout(d time.Duration) time.Duration {
	if d <= 0 {
		return 5 * time.Minute
	}
	return d
}
