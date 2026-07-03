package collector

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeMinimalArchive writes a .tar.gz with a tiny manifest.json and a
// dummy file so InspectArchive has something to chew on.
func makeMinimalArchive(t *testing.T, dest string, manifestJSON string, extra string) {
	t.Helper()
	f, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gzw := gzip.NewWriter(f)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	header := &tar.Header{Name: "extras/manifest.json", Mode: 0o644, Size: int64(len(manifestJSON))}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	if extra != "" {
		parts := strings.SplitN(extra, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("bad extra: %q", extra)
		}
		name, body := parts[0], parts[1]
		h := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInspectArchive_listsFilesAndReadsManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.tar.gz")
	makeMinimalArchive(t, path, `{"groot_version":"test","cluster":{"cluster":"kind"}}`, "nodes/n1.log=line1\nline2\n")

	info, err := InspectArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.FileCount == 0 {
		t.Fatal("expected > 0 files")
	}
	if info.ArchiveSize <= 0 {
		t.Fatalf("expected archive size > 0, got %d", info.ArchiveSize)
	}
	if info.ManifestJSON == "" {
		t.Fatal("expected manifest json to be parsed and surfaced")
	}
	if !strings.Contains(info.ManifestJSON, "kind") {
		t.Fatalf("manifest not echoed back: %q", info.ManifestJSON)
	}
	if len(info.Files) < 2 {
		t.Fatalf("expected at least 2 file entries, got %d", len(info.Files))
	}
}

func TestInspectArchive_missingFile(t *testing.T) {
	_, err := InspectArchive(filepath.Join(t.TempDir(), "nope.tar.gz"))
	if err == nil {
		t.Fatal("expected error for missing archive")
	}
}

func TestInspectArchive_badGzip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.tar.gz")
	if err := os.WriteFile(path, []byte("not-a-gzip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectArchive(path); err == nil {
		t.Fatal("expected gzip error")
	}
}

func TestInspectArchive_handlesUnparseableManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.tar.gz")
	makeMinimalArchive(t, path, "this is not json", "")

	info, err := InspectArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.ParseErr == "" {
		t.Fatalf("expected ParseErr to be populated, got %+v", info)
	}
}

func TestInspectArchive_compressedTampered(t *testing.T) {
	// Construct an archive that is gzip-valid but tar-malformed (truncated).
	dir := t.TempDir()
	path := filepath.Join(dir, "trunc.tar.gz")
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	gzw.Write([]byte("not a tar stream"))
	gzw.Close()
	os.WriteFile(path, buf.Bytes(), 0o644)

	if _, err := InspectArchive(path); err == nil {
		t.Fatal("expected tar read error")
	}
}
