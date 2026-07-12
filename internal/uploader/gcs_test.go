package uploader

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
)

func startFakeGCS(t *testing.T, bucket string) *fakestorage.Server {
	t.Helper()
	server, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		Scheme:     "http",
		Host:       "127.0.0.1",
		PublicHost: "127.0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: bucket})
	t.Cleanup(server.Stop)

	u, err := url.Parse(server.URL())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("STORAGE_EMULATOR_HOST", u.Host)
	return server
}

func TestGCSClientOptions_emulatorHost(t *testing.T) {
	t.Setenv("STORAGE_EMULATOR_HOST", "127.0.0.1:4443")
	opts := gcsClientOptions()
	if len(opts) != 2 {
		t.Fatalf("opts=%d want 2 (endpoint + no auth)", len(opts))
	}

	t.Setenv("STORAGE_EMULATOR_HOST", "")
	if got := gcsClientOptions(); got != nil {
		t.Fatalf("expected nil without emulator, got %v", got)
	}
}

func TestGCSUploader_upload_emulator(t *testing.T) {
	const bucket = "groot-test"
	server := startFakeGCS(t, bucket)

	dir := t.TempDir()
	archive := filepath.Join(dir, "capture.tar.gz")
	body := []byte("gzip-bytes")
	if err := os.WriteFile(archive, body, 0o644); err != nil {
		t.Fatal(err)
	}

	up := newGCSUploader(config.GCSUploadCfg{
		Enabled:     true,
		Bucket:      bucket,
		KeyPrefix:   "archives/",
		ContentType: "application/gzip",
		Metadata:    map[string]string{"env": "test"},
	}, 0)
	res, err := up.Upload(context.Background(), archive, collector.Summary{RunID: "run-abc"})
	if err != nil {
		t.Fatal(err)
	}
	wantKey := "archives/capture.tar.gz"
	if res.URI != "gs://"+bucket+"/"+wantKey || res.Key != wantKey || res.Size != int64(len(body)) {
		t.Fatalf("result=%+v", res)
	}

	obj, err := server.GetObject(bucket, wantKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(obj.Content) != string(body) {
		t.Fatalf("content=%q", obj.Content)
	}
	if obj.ContentType != "application/gzip" {
		t.Fatalf("content-type=%q", obj.ContentType)
	}
	if obj.Metadata["env"] != "test" || obj.Metadata["run_id"] != "run-abc" {
		t.Fatalf("metadata=%v", obj.Metadata)
	}
}

func TestGCSUploader_contextCanceled(t *testing.T) {
	const bucket = "groot-cancel"
	startFakeGCS(t, bucket)

	dir := t.TempDir()
	archive := filepath.Join(dir, "capture.tar.gz")
	if err := os.WriteFile(archive, []byte("gzip-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	up := newGCSUploader(config.GCSUploadCfg{Enabled: true, Bucket: bucket}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := up.Upload(ctx, archive, collector.Summary{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGCSUploader_archiveNotFound(t *testing.T) {
	const bucket = "groot-missing-file"
	startFakeGCS(t, bucket)

	up := newGCSUploader(config.GCSUploadCfg{Enabled: true, Bucket: bucket}, 0)
	_, err := up.Upload(context.Background(), "/nonexistent/capture.tar.gz", collector.Summary{})
	if err == nil || !strings.Contains(err.Error(), "open archive") {
		t.Fatalf("err=%v", err)
	}
}
