package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BuildInfo is injected from the CLI for archive manifests.
type BuildInfo struct {
	Version string
	Commit  string
	Branch  string
	Date    string
}

type manifestCluster struct {
	Context string `json:"context"`
	Cluster string `json:"cluster"`
	User    string `json:"user"`
	Server  string `json:"server"`
}

type manifestJobs struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type captureManifest struct {
	GrootVersion    string          `json:"groot_version"`
	GrootCommit     string          `json:"groot_commit,omitempty"`
	CollectedAt     string          `json:"collected_at"`
	DurationSeconds float64         `json:"duration_seconds"`
	SessionBase     string          `json:"session_base"`
	ArchiveBasename string          `json:"archive_basename"`
	FilePrefix      string          `json:"file_prefix"`
	Cluster         manifestCluster `json:"cluster"`
	Jobs            manifestJobs    `json:"jobs"`
	Paths           []string        `json:"paths"`
}

// SetBuildInfo records CLI build metadata for manifests.
func (s *Service) SetBuildInfo(version, commit, branch, date string) {
	s.buildInfo = BuildInfo{Version: version, Commit: commit, Branch: branch, Date: date}
}

func (s *Service) writeManifest(
	ctx context.Context,
	captureDir, sessionBase, archiveBasename string,
	summary Summary,
) error {
	meta, err := s.ReadKubeMetadata(ctx)
	if err != nil {
		return err
	}

	paths, err := listCaptureRelPaths(captureDir)
	if err != nil {
		return err
	}

	ver := strings.TrimSpace(s.buildInfo.Version)
	if ver == "" {
		ver = "dev"
	}

	m := captureManifest{
		GrootVersion:    ver,
		GrootCommit:     strings.TrimSpace(s.buildInfo.Commit),
		CollectedAt:     time.Now().UTC().Format(time.RFC3339),
		DurationSeconds: summary.Duration.Seconds(),
		SessionBase:     sessionBase,
		ArchiveBasename: archiveBasename,
		FilePrefix:      strings.TrimSpace(s.cfg.FilePrefix),
		Cluster: manifestCluster{
			Context: emptyAsUnknown(meta.Context),
			Cluster: emptyAsUnknown(meta.Cluster),
			User:    emptyAsUnknown(meta.User),
			Server:  emptyAsUnknown(meta.Server),
		},
		Jobs: manifestJobs{
			Total:   summary.Total,
			Success: summary.Success,
			Failed:  summary.Failed,
		},
		Paths: paths,
	}

	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	target := filepath.Join(captureDir, "extras", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	if err := os.WriteFile(target, body, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func listCaptureRelPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}
