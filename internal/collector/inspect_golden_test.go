package collector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/archive"
)

// Golden-style inspect regression without a live cluster (ROADMAP 1.0.0 #87).
func TestInspectArchive_goldenFixture(t *testing.T) {
	src := t.TempDir()
	manifest := filepath.Join(src, "extras", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"groot_version":          "1.0.0-test",
		"archive_layout_version": ArchiveLayoutVersion,
		"config_version":         1,
		"session_base":           "groot-capture-20260703-120000",
		"archive_basename":       "groot-capture-20260703-120000-demo",
		"paths":                  []string{"extras/manifest.json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, body, 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "inspect-golden.tar.gz")
	if err := archive.DirToTarGz(src, archivePath); err != nil {
		t.Fatal(err)
	}

	info, err := InspectArchive(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.FileCount < 1 {
		t.Fatalf("file_count=%d", info.FileCount)
	}
	if info.ManifestJSON == "" {
		t.Fatal("expected manifest in archive")
	}
	if info.ParseErr != "" {
		t.Fatalf("ParseErr=%q", info.ParseErr)
	}
	// DirToTarGz prefixes members with the session directory basename; inventory
	// must still surface extras/manifest.json via arcread suffix lookup.
	foundManifest := false
	for _, line := range info.Files {
		if strings.Contains(line, "extras/manifest.json (") {
			foundManifest = true
			break
		}
	}
	if !foundManifest {
		t.Fatalf("session-prefixed inventory missing extras/manifest.json: %v", info.Files)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(info.ManifestJSON), &m); err != nil {
		t.Fatal(err)
	}
	if m["archive_layout_version"] != float64(ArchiveLayoutVersion) {
		t.Fatalf("manifest archive_layout_version=%v", m["archive_layout_version"])
	}
}
