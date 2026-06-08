package uploader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/groot/pkg/collector"
	"github.com/hrodrig/groot/pkg/config"
)

func TestShouldUpload(t *testing.T) {
	if ShouldUpload(config.Config{}) {
		t.Fatal("expected false")
	}
	cfg := config.Config{Upload: config.UploadCfg{Enabled: true, S3: config.S3UploadCfg{Enabled: true, Bucket: "b"}}}
	if !ShouldUpload(cfg) {
		t.Fatal("expected true")
	}
}

func TestObjectKey(t *testing.T) {
	if got := objectKey("", "/tmp/out/foo.tar.gz"); got != "foo.tar.gz" {
		t.Fatalf("got %q", got)
	}
	if got := objectKey("archives/", "/tmp/out/foo.tar.gz"); got != "archives/foo.tar.gz" {
		t.Fatalf("got %q", got)
	}
}

func TestFanOut_Upload_continueOnError(t *testing.T) {
	f := &FanOut{
		uploaders: []Uploader{
			stubUploader{provider: "a", err: errStub("a fail")},
			stubUploader{provider: "b", res: &Result{Provider: "b", URI: "s3://b/k"}},
		},
		continueOnError: true,
	}
	out := f.Upload(context.Background(), "x.tar.gz", collector.Summary{})
	if len(out) != 2 {
		t.Fatalf("outcomes=%d", len(out))
	}
	if out[0].Err == nil || out[1].Err != nil || out[1].Result == nil {
		t.Fatalf("%+v", out)
	}
}

func TestS3Uploader_upload_httptest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/my-bucket/") {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("ETag", `"etag1"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dir := t.TempDir()
	archive := filepath.Join(dir, "capture.tar.gz")
	if err := os.WriteFile(archive, []byte("gzip-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")

	up := newS3Uploader(config.S3UploadCfg{
		Enabled:  true,
		Bucket:   "my-bucket",
		Region:   "us-east-1",
		Endpoint: srv.URL,
	}, 0)
	res, err := up.Upload(context.Background(), archive, collector.Summary{})
	if err != nil {
		t.Fatal(err)
	}
	if res.URI != "s3://my-bucket/capture.tar.gz" || res.ETag != `"etag1"` {
		t.Fatalf("%+v", res)
	}
}

type stubUploader struct {
	provider string
	res      *Result
	err      error
}

func (s stubUploader) Provider() string { return s.provider }

func (s stubUploader) Upload(context.Context, string, collector.Summary) (*Result, error) {
	return s.res, s.err
}

type errStub string

func (e errStub) Error() string { return string(e) }

func TestFormatResult(t *testing.T) {
	line := FormatResult(&Result{Provider: "s3", URI: "s3://b/k", Size: 9, ETag: `"e"`})
	if !strings.Contains(line, "s3://b/k") || !strings.Contains(line, `etag="e"`) {
		t.Fatalf("%q", line)
	}
}

func TestGCSUploader_missingBucket(t *testing.T) {
	up := newGCSUploader(config.GCSUploadCfg{Enabled: true}, 0)
	_, err := up.Upload(context.Background(), "x.tar.gz", collector.Summary{})
	if err == nil || !strings.Contains(err.Error(), "bucket") {
		t.Fatalf("err=%v", err)
	}
}

func TestS3Uploader_missingBucket(t *testing.T) {
	up := newS3Uploader(config.S3UploadCfg{Enabled: true}, 0)
	_, err := up.Upload(context.Background(), "x.tar.gz", collector.Summary{})
	if err == nil || !strings.Contains(err.Error(), "bucket") {
		t.Fatalf("err=%v", err)
	}
}

func TestNewFanOut_buildsProviders(t *testing.T) {
	cfg := config.Config{Upload: config.UploadCfg{
		Enabled:         true,
		ContinueOnError: true,
		S3:              config.S3UploadCfg{Enabled: true, Bucket: "b"},
		GCS:             config.GCSUploadCfg{Enabled: true, Bucket: "g"},
	}}
	if !ShouldUpload(cfg) {
		t.Fatal("expected upload enabled")
	}
	f := NewFanOut(cfg)
	if len(f.Upload(context.Background(), "missing.tar.gz", collector.Summary{})) != 2 {
		t.Fatal("expected two provider outcomes")
	}
}
