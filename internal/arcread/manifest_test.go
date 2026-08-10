package arcread_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrodrig/groot/internal/arcread"
)

const goldenManifestLayout1 = `{
  "groot_version": "1.0.6",
  "groot_commit": "abc123",
  "config_version": 2,
  "archive_layout_version": 1,
  "run_id": "run-1",
  "archive_sha256": "deadbeef",
  "collected_at": "2026-01-02T15:04:05Z",
  "duration_seconds": 2.5,
  "session_base": "groot-capture-20260102-150405",
  "archive_basename": "groot-capture-20260102-150405-cluster",
  "file_prefix": "groot-capture",
  "cluster": {
    "context": "kind-kind",
    "cluster": "kind",
    "user": "kind-user",
    "server": "https://127.0.0.1:6443"
  },
  "jobs": {
    "total": 3,
    "success": 2,
    "failed": 1
  },
  "paths": [
    "extras/kubeconfig.txt",
    "extras/manifest.json"
  ]
}`

func TestDecodeManifest_goldenLayout1(t *testing.T) {
	m, err := arcread.DecodeManifest([]byte(goldenManifestLayout1))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	if m.ArchiveLayoutVersion != arcread.ArchiveLayoutVersion {
		t.Fatalf("archive_layout_version=%d want %d", m.ArchiveLayoutVersion, arcread.ArchiveLayoutVersion)
	}
	if m.GrootVersion != "1.0.6" || m.GrootCommit != "abc123" || m.ConfigVersion != 2 {
		t.Fatalf("identity fields: %+v", m)
	}
	if m.RunID != "run-1" || m.ArchiveSHA256 != "deadbeef" {
		t.Fatalf("run fields: %+v", m)
	}
	if m.SessionBase == "" || m.ArchiveBasename == "" || m.FilePrefix != "groot-capture" {
		t.Fatalf("path fields: %+v", m)
	}
	if m.Cluster.Cluster != "kind" || m.Cluster.Server == "" {
		t.Fatalf("cluster: %+v", m.Cluster)
	}
	if m.Jobs.Total != 3 || m.Jobs.Success != 2 || m.Jobs.Failed != 1 {
		t.Fatalf("jobs: %+v", m.Jobs)
	}
	if len(m.Paths) != 2 {
		t.Fatalf("paths=%v", m.Paths)
	}
}

func TestManifest_roundTripCollectShapedEncoder(t *testing.T) {
	// Encode with the same exported type collect writeManifest must populate.
	in := arcread.Manifest{
		GrootVersion:         "1.0.6",
		GrootCommit:          "abc123",
		ConfigVersion:        2,
		ArchiveLayoutVersion: arcread.ArchiveLayoutVersion,
		RunID:                "run-1",
		ArchiveSHA256:        "deadbeef",
		CollectedAt:          "2026-01-02T15:04:05Z",
		DurationSeconds:      2.5,
		SessionBase:          "groot-capture-20260102-150405",
		ArchiveBasename:      "groot-capture-20260102-150405-cluster",
		FilePrefix:           "groot-capture",
		Cluster: arcread.ManifestCluster{
			Context: "kind-kind",
			Cluster: "kind",
			User:    "kind-user",
			Server:  "https://127.0.0.1:6443",
		},
		Jobs: arcread.ManifestJobs{Total: 3, Success: 2, Failed: 1},
		Paths: []string{
			"extras/kubeconfig.txt",
			"extras/manifest.json",
		},
	}
	body, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "session")
	if err := os.MkdirAll(filepath.Join(src, "extras"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "extras", "manifest.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	// Pack with a tiny helper tar (bare prefix) so Archive.Manifest can load it.
	path := filepath.Join(dir, "m.tar.gz")
	writeBareArchive(t, path, string(body), "", "")

	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	out, err := arc.Manifest()
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if out.ArchiveLayoutVersion != arcread.ArchiveLayoutVersion {
		t.Fatalf("layout=%d", out.ArchiveLayoutVersion)
	}
	if out.GrootVersion != in.GrootVersion || out.Cluster.Cluster != in.Cluster.Cluster || out.Jobs.Failed != in.Jobs.Failed {
		t.Fatalf("round-trip loss: got %+v want %+v", out, in)
	}
	if len(out.Paths) != len(in.Paths) {
		t.Fatalf("paths lost: %v", out.Paths)
	}
}
