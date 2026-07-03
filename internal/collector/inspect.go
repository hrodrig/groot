package collector

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// InspectInfo summarizes the contents of a completed groot archive (.tar.gz).
// All fields are deducted from the archive itself—no cluster access required.
type InspectInfo struct {
	ArchivePath  string   `json:"archive_path"`
	ArchiveSize  int64    `json:"archive_size_bytes"`
	FileCount    int      `json:"file_count"`
	Files        []string `json:"files"`
	ManifestJSON string   `json:"manifest_json,omitempty"`
	ParseErr     string   `json:"parse_err,omitempty"`
}

// InspectArchive opens an existing .tar.gz produced by `groot collect`,
// lists its contents, extracts extras/manifest.json if present, and returns
// the summary. This is the "minimum" version from ROADMAP #31 (0.9.x).
//
// The archive path is resolved relative to the current working directory.
func InspectArchive(archivePath string) (InspectInfo, error) {
	abs, err := filepath.Abs(archivePath)
	if err != nil {
		return InspectInfo{}, fmt.Errorf("resolve path: %w", err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return InspectInfo{}, fmt.Errorf("stat %s: %w", abs, err)
	}

	f, err := os.Open(abs)
	if err != nil {
		return InspectInfo{}, fmt.Errorf("open %s: %w", abs, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return InspectInfo{}, fmt.Errorf("gzip reader: %w", err)
	}
	defer gzr.Close()

	info := InspectInfo{
		ArchivePath: abs,
		ArchiveSize: fi.Size(),
	}

	tarReader := tar.NewReader(gzr)
	for {
		hdr, terr := tarReader.Next()
		if terr == io.EOF {
			break
		}
		if terr != nil {
			return InspectInfo{}, fmt.Errorf("tar read: %w", terr)
		}
		if hdr.Typeflag == tar.TypeReg && hdr.Name != "" {
			info.Files = append(info.Files, fmt.Sprintf("%s (%d bytes)", hdr.Name, hdr.Size))
			info.FileCount++
			// Attempt to read manifest.json for a pretty-print summary.
			if strings.HasSuffix(hdr.Name, "extras/manifest.json") {
				buf, rerr := io.ReadAll(tarReader)
				if rerr != nil {
					info.ParseErr = fmt.Sprintf("read manifest: %v", rerr)
					continue
				}
				// Compact JSON, but only validate it parses.
				var raw any
				if jerr := json.Unmarshal(buf, &raw); jerr != nil {
					info.ParseErr = fmt.Sprintf("manifest parse: %v", jerr)
					continue
				}
				info.ManifestJSON = string(buf)
			}
		}
	}
	return info, nil
}
