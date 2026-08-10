package arcread

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ArchiveLayoutVersion is the groot capture layout encoded in extras/manifest.json.
const ArchiveLayoutVersion = 1

// ManifestCluster is the cluster identity block in extras/manifest.json.
type ManifestCluster struct {
	Context string `json:"context"`
	Cluster string `json:"cluster"`
	User    string `json:"user"`
	Server  string `json:"server"`
}

// ManifestJobs is the job counters block in extras/manifest.json.
type ManifestJobs struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// Manifest is the typed extras/manifest.json schema for archive_layout_version 1.
type Manifest struct {
	GrootVersion         string          `json:"groot_version"`
	GrootCommit          string          `json:"groot_commit,omitempty"`
	ConfigVersion        int             `json:"config_version,omitempty"`
	ArchiveLayoutVersion int             `json:"archive_layout_version"`
	RunID                string          `json:"run_id,omitempty"`
	ArchiveSHA256        string          `json:"archive_sha256,omitempty"`
	CollectedAt          string          `json:"collected_at"`
	DurationSeconds      float64         `json:"duration_seconds"`
	SessionBase          string          `json:"session_base"`
	ArchiveBasename      string          `json:"archive_basename"`
	FilePrefix           string          `json:"file_prefix"`
	Cluster              ManifestCluster `json:"cluster"`
	Jobs                 ManifestJobs    `json:"jobs"`
	Paths                []string        `json:"paths"`
}

// DecodeManifest parses extras/manifest.json bytes into a typed Manifest.
func DecodeManifest(raw []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrManifestParse, err)
	}
	return m, nil
}

// Manifest returns the typed decode of extras/manifest.json.
func (a *Archive) Manifest() (Manifest, error) {
	raw, err := a.ManifestRaw()
	if err != nil {
		return Manifest{}, err
	}
	return DecodeManifest(raw)
}

// ManifestRaw returns the raw bytes of extras/manifest.json (Pass-1 cache or reopen).
func (a *Archive) ManifestRaw() ([]byte, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if a.manifestCache != nil {
		out := make([]byte, len(a.manifestCache))
		copy(out, a.manifestCache)
		return out, nil
	}
	meta, ok := a.LookupSuffix("extras/manifest.json")
	if !ok {
		return nil, ErrManifestMissing
	}
	return a.ReadMember(meta.Name)
}

func isManifestName(name string) bool {
	return name == "extras/manifest.json" || strings.HasSuffix(name, "/extras/manifest.json")
}
