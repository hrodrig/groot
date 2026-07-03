package uploader

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
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

func TestShouldUpload_sftp(t *testing.T) {
	if ShouldUpload(config.Config{}) {
		t.Fatal("expected false")
	}
	cfg := config.Config{Upload: config.UploadCfg{
		Enabled: true,
		SFTP:    config.SFTPUploadCfg{Enabled: true, Host: "h", Port: 22},
	}}
	if !ShouldUpload(cfg) {
		t.Fatal("expected true")
	}
}

func TestSFTPUploader_missingHost(t *testing.T) {
	up := newSFTPUploader(config.SFTPUploadCfg{Enabled: true}, 0)
	_, err := up.Upload(context.Background(), "x.tar.gz", collector.Summary{})
	if err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("err=%v", err)
	}
}

func TestSFTPUploader_missingIdentity(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "capture.tar.gz")
	if err := os.WriteFile(archive, []byte("gzip-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	up := newSFTPUploader(config.SFTPUploadCfg{
		Enabled: true,
		Host:    "example.com",
		Port:    22,
		User:    "test",
	}, 0)
	_, err := up.Upload(context.Background(), archive, collector.Summary{})
	if err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("err=%v", err)
	}
}

func TestFanOut_withSFTP(t *testing.T) {
	cfg := config.Config{Upload: config.UploadCfg{
		Enabled:         true,
		ContinueOnError: true,
		S3:              config.S3UploadCfg{Enabled: true, Bucket: "b"},
		SFTP:            config.SFTPUploadCfg{Enabled: true, Host: "h", Port: 22},
	}}
	if !ShouldUpload(cfg) {
		t.Fatal("expected upload enabled")
	}
	f := NewFanOut(cfg)
	if len(f.uploaders) != 2 {
		t.Fatalf("expected 2 uploaders, got %d", len(f.uploaders))
	}
	out := f.Upload(context.Background(), "missing.tar.gz", collector.Summary{})
	if len(out) != 2 {
		t.Fatalf("expected 2 outcomes, got %d", len(out))
	}
	// S3 fails (missing bucket), SFTP fails (missing identity), continueOnError=true
	if out[0].Err == nil || out[1].Err == nil {
		t.Fatal("expected both to fail")
	}
}

func TestSFTPUploader_successPath(t *testing.T) {
	// Generate test key pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}

	// Start SSH server
	srvCfg := &ssh.ServerConfig{
		NoClientAuth: false,
		PublicKeyCallback: func(conn ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
	}
	srvCfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		sconn, chans, reqs, err := ssh.NewServerConn(conn, srvCfg)
		if err != nil {
			conn.Close()
			return
		}
		defer sconn.Close()
		go ssh.DiscardRequests(reqs)
		// Reject all channels (no SFTP subsystem)
		for ch := range chans {
			ch.Reject(ssh.UnknownChannelType, "no sftp subsystem")
		}
		close(done)
	}()

	// Write test key file
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_test")
	privBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(keyPath, privBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	// Write archive
	archive := filepath.Join(dir, "capture.tar.gz")
	if err := os.WriteFile(archive, []byte("gzip-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	up := newSFTPUploader(config.SFTPUploadCfg{
		Enabled:              true,
		Host:                 host,
		Port:                 port,
		User:                 "test",
		IdentityFile:         keyPath,
		AllowInsecureHostKey: true,
	}, 0)
	_, err = up.Upload(context.Background(), archive, collector.Summary{})
	if err == nil {
		t.Fatal("expected error (no sftp subsystem on server)")
	}
	if !strings.Contains(err.Error(), "sftp") {
		t.Fatalf("unexpected error: %v", err)
	}
	<-done
}

func TestSFTPUploader_archiveNotFound(t *testing.T) {
	up := newSFTPUploader(config.SFTPUploadCfg{
		Enabled:      true,
		Host:         "example.com",
		Port:         22,
		User:         "test",
		IdentityFile: "/nonexistent/key",
	}, 0)
	_, err := up.Upload(context.Background(), "/nonexistent/archive.tar.gz", collector.Summary{})
	if err == nil || !strings.Contains(err.Error(), "open archive") {
		t.Fatalf("err=%v", err)
	}
}

func TestSFTPUploader_knownHostsMismatch(t *testing.T) {
	// Generate server key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}

	srvCfg := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	srvCfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Do SSH handshake so client sees the real host key
		sconn, _, _, _ := ssh.NewServerConn(conn, srvCfg)
		if sconn != nil {
			sconn.Close()
		} else {
			conn.Close()
		}
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	// Write a known_hosts file with a WRONG key
	dir := t.TempDir()
	wrongKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	wrongPub, _ := ssh.NewPublicKey(&wrongKey.PublicKey)
	knownHostsPath := filepath.Join(dir, "known_hosts")
	knownHostsLine := knownhosts.Line([]string{host}, wrongPub) + "\n"
	if err := os.WriteFile(knownHostsPath, []byte(knownHostsLine), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write identity
	identityPath := filepath.Join(dir, "id_test")
	privBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(identityPath, privBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	// Write archive
	archive := filepath.Join(dir, "capture.tar.gz")
	if err := os.WriteFile(archive, []byte("gzip-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	up := newSFTPUploader(config.SFTPUploadCfg{
		Enabled:        true,
		Host:           host,
		Port:           port,
		User:           "test",
		IdentityFile:   identityPath,
		KnownHostsFile: knownHostsPath,
	}, 0)
	_, err = up.Upload(context.Background(), archive, collector.Summary{})
	if err == nil {
		t.Fatal("expected host key mismatch error")
	}
	if !strings.Contains(err.Error(), "host key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
