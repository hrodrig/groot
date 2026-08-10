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

	"github.com/hrodrig/groot/internal/arcread"
)

// BuildInfo is injected from the CLI for archive manifests.
type BuildInfo struct {
	Version string
	Commit  string
	Branch  string
	Date    string
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
	cluster := s.resolveClusterName(ctx)
	if strings.TrimSpace(meta.Server) == "" {
		if s.restConfig != nil {
			meta.Server = strings.TrimSpace(s.restConfig.Host)
		} else {
			meta.Server = restHostFromKubeconfig(s.cfg.Kubeconfig)
		}
	}

	paths, err := s.captureManifestPaths(captureDir)
	if err != nil {
		return err
	}

	ver := strings.TrimSpace(s.buildInfo.Version)
	if ver == "" {
		ver = "dev"
	}

	m := arcread.Manifest{
		GrootVersion:         ver,
		GrootCommit:          strings.TrimSpace(s.buildInfo.Commit),
		ConfigVersion:        s.cfg.ConfigVersion,
		ArchiveLayoutVersion: ArchiveLayoutVersion,
		RunID:                s.RunID,
		ArchiveSHA256:        s.archiveSHA256,
		CollectedAt:          time.Now().UTC().Format(time.RFC3339),
		DurationSeconds:      summary.Duration.Seconds(),
		SessionBase:          sessionBase,
		ArchiveBasename:      archiveBasename,
		FilePrefix:           strings.TrimSpace(s.cfg.FilePrefix),
		Cluster: arcread.ManifestCluster{
			Context: emptyAsUnknown(meta.Context),
			Cluster: emptyAsUnknown(cluster),
			User:    emptyAsUnknown(meta.User),
			Server:  emptyAsUnknown(meta.Server),
		},
		Jobs: arcread.ManifestJobs{
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

func (s *Service) captureManifestPaths(captureDir string) ([]string, error) {
	if len(s.manifestPaths) > 0 {
		return s.manifestPaths, nil
	}
	paths, err := listCaptureRelPaths(captureDir)
	if err != nil {
		return nil, err
	}
	s.manifestPaths = paths
	return paths, nil
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
