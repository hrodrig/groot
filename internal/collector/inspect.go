package collector

import (
	"fmt"

	"github.com/hrodrig/groot/internal/arcread"
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
// Inventory uses the shared offline reader (arcread); UX stays inventory-only.
func InspectArchive(archivePath string) (InspectInfo, error) {
	arc, err := arcread.Open(archivePath)
	if err != nil {
		return InspectInfo{}, err
	}
	defer arc.Close()

	info := InspectInfo{
		ArchivePath: arc.Path(),
		ArchiveSize: arc.Size(),
	}
	for _, m := range arc.Members() {
		info.Files = append(info.Files, fmt.Sprintf("%s (%d bytes)", m.Name, m.Size))
		info.FileCount++
	}

	raw, err := arc.ManifestRaw()
	if err != nil {
		// Missing manifest is not a hard error for inventory-only inspect.
		return info, nil
	}
	if _, perr := arcread.DecodeManifest(raw); perr != nil {
		info.ParseErr = fmt.Sprintf("manifest parse: %v", perr)
		return info, nil
	}
	info.ManifestJSON = string(raw)
	return info, nil
}
