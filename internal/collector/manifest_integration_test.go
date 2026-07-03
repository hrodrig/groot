package collector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hrodrig/groot/internal/config"
	"github.com/hrodrig/groot/internal/kubetest"
)

func TestWriteManifest(t *testing.T) {
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	root := t.TempDir()
	s := New(config.Config{FilePrefix: "groot-capture", Kubeconfig: kc})
	s.SetBuildInfo("0.4.0", "abc", "develop", "now")
	if err := os.MkdirAll(filepath.Join(root, "extras"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "extras", "kubeconfig.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	summary := Summary{Total: 3, Success: 2, Failed: 1, Duration: 2 * time.Second}
	if err := s.writeManifest(t.Context(), root, "groot-capture-20260102-150405", "groot-capture-20260102-150405-cluster", summary); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "extras", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m captureManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.GrootVersion != "0.4.0" || m.Jobs.Total != 3 || len(m.Paths) == 0 {
		t.Fatalf("%+v", m)
	}
	if m.ArchiveLayoutVersion != ArchiveLayoutVersion {
		t.Fatalf("archive_layout_version=%d want %d", m.ArchiveLayoutVersion, ArchiveLayoutVersion)
	}
}
