package arcread_test

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/archive"
	"github.com/hrodrig/groot/internal/arcread"
	"github.com/hrodrig/groot/internal/collector"
)

func writeBareArchive(t *testing.T, dest, manifestJSON, extraName, extraBody string) {
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

	mh := &tar.Header{Name: "extras/manifest.json", Mode: 0o644, Size: int64(len(manifestJSON))}
	if err := tw.WriteHeader(mh); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(manifestJSON)); err != nil {
		t.Fatal(err)
	}
	if extraName != "" {
		eh := &tar.Header{Name: extraName, Mode: 0o644, Size: int64(len(extraBody))}
		if err := tw.WriteHeader(eh); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(extraBody)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOpen_indexesRegularFilesOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bare.tar.gz")
	manifest := `{"groot_version":"test","archive_layout_version":1,"collected_at":"2026-01-02T15:04:05Z","duration_seconds":1,"session_base":"s","archive_basename":"s","file_prefix":"groot-capture","cluster":{"context":"c","cluster":"kind","user":"u","server":"https://x"},"jobs":{"total":1,"success":1,"failed":0},"paths":["extras/manifest.json"]}`
	writeBareArchive(t, path, manifest, "nodes/n1.log", "hello-bytes")

	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	members := arc.Members()
	if len(members) != 2 {
		t.Fatalf("Members=%d want 2 regular files, got %+v", len(members), members)
	}
	for _, m := range members {
		if m.Name == "" || m.Size < 0 {
			t.Fatalf("bad member meta: %+v", m)
		}
	}
	if _, ok := arc.LookupSuffix("extras/manifest.json"); !ok {
		t.Fatal("LookupSuffix(extras/manifest.json) miss")
	}
}

func TestManifest_typedDecodeLayoutVersion1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bare.tar.gz")
	manifest := `{"groot_version":"test","archive_layout_version":1,"collected_at":"2026-01-02T15:04:05Z","duration_seconds":1.5,"session_base":"s","archive_basename":"s","file_prefix":"groot-capture","cluster":{"context":"c","cluster":"kind","user":"u","server":"https://x"},"jobs":{"total":1,"success":1,"failed":0},"paths":["extras/manifest.json"]}`
	writeBareArchive(t, path, manifest, "", "")

	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	m, err := arc.Manifest()
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if m.ArchiveLayoutVersion != arcread.ArchiveLayoutVersion {
		t.Fatalf("archive_layout_version=%d want %d", m.ArchiveLayoutVersion, arcread.ArchiveLayoutVersion)
	}
	if m.GrootVersion != "test" || m.Cluster.Cluster != "kind" {
		t.Fatalf("typed fields wrong: %+v", m)
	}
}

func TestReadMember_twoPassExactBytes(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "session")
	if err := os.MkdirAll(filepath.Join(src, "extras"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("exact-member-bytes\n")
	if err := os.WriteFile(filepath.Join(src, "extras", "sample.txt"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"groot_version":"test","archive_layout_version":1,"collected_at":"2026-01-02T15:04:05Z","duration_seconds":0,"session_base":"session","archive_basename":"session","file_prefix":"groot-capture","cluster":{},"jobs":{},"paths":[]}`
	if err := os.WriteFile(filepath.Join(src, "extras", "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "session.tar.gz")
	if err := archive.DirToTarGz(src, path); err != nil {
		t.Fatal(err)
	}

	beforeEntries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	meta, ok := arc.LookupSuffix("extras/sample.txt")
	if !ok {
		t.Fatal("LookupSuffix(extras/sample.txt) miss")
	}
	got, err := arc.ReadMember(meta.Name)
	if err != nil {
		t.Fatalf("ReadMember: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("ReadMember bytes=%q want %q", got, body)
	}

	afterEntries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterEntries) != len(beforeEntries) {
		t.Fatalf("extract tree created under %s: before=%d after=%d", dir, len(beforeEntries), len(afterEntries))
	}
}

func TestInspectArchive_inventoryViaArcread(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.tar.gz")
	manifest := `{"groot_version":"test","archive_layout_version":1,"cluster":{"cluster":"kind"}}`
	writeBareArchive(t, path, manifest, "nodes/n1.log", "line1\nline2\n")

	info, err := collector.InspectArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.FileCount != 2 {
		t.Fatalf("FileCount=%d want 2", info.FileCount)
	}
	if info.ArchiveSize <= 0 {
		t.Fatalf("ArchiveSize=%d", info.ArchiveSize)
	}
	if info.ManifestJSON == "" || !strings.Contains(info.ManifestJSON, "kind") {
		t.Fatalf("ManifestJSON=%q", info.ManifestJSON)
	}
	if info.ParseErr != "" {
		t.Fatalf("ParseErr=%q", info.ParseErr)
	}
	found := false
	for _, line := range info.Files {
		if strings.HasPrefix(line, "nodes/n1.log (") && strings.HasSuffix(line, " bytes)") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Files missing inventory line: %v", info.Files)
	}

	badPath := filepath.Join(dir, "missing.tar.gz")
	if _, err := collector.InspectArchive(badPath); err == nil {
		t.Fatal("expected open failure for missing archive")
	}

	broken := filepath.Join(dir, "broken.tar.gz")
	writeBareArchive(t, broken, "not-json", "", "")
	info2, err := collector.InspectArchive(broken)
	if err != nil {
		t.Fatal(err)
	}
	if info2.ParseErr == "" {
		t.Fatalf("expected ParseErr, got %+v", info2)
	}
}
