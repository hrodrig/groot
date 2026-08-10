package analyze

import (
	"errors"
	"fmt"

	"github.com/hrodrig/groot/internal/arcread"
)

// Run analyzes an already-open archive. It does not Close the archive.
// Missing optional members become Notes; only catastrophic I/O returns an error.
func Run(arc *arcread.Archive) (Report, error) {
	if arc == nil {
		return Report{}, fmt.Errorf("analyze: nil archive")
	}

	rep := Report{
		ArchivePath: arc.Path(),
		Hints:       []Hint{},
	}

	if m, err := arc.Manifest(); err != nil {
		switch {
		case errors.Is(err, arcread.ErrManifestMissing):
			rep.Notes = append(rep.Notes, Note{
				Code:    "member_missing",
				Message: "extras/manifest.json not found",
				Path:    "extras/manifest.json",
			})
		case errors.Is(err, arcread.ErrManifestParse):
			rep.Notes = append(rep.Notes, Note{
				Code:    "manifest_parse",
				Message: fmt.Sprintf("manifest parse degraded: %v", err),
				Path:    "extras/manifest.json",
			})
		default:
			// Unexpected read failure while resolving the manifest is catastrophic.
			return Report{}, fmt.Errorf("read manifest: %w", err)
		}
	} else {
		rep.RunID = m.RunID
		rep.ArchiveSHA256 = m.ArchiveSHA256
		rep.ArchiveLayoutVersion = m.ArchiveLayoutVersion
	}

	ev, err := loadEvidence(arc, &rep.Notes)
	if err != nil {
		return Report{}, err
	}

	rep.Hints = runHeuristics(ev)
	sortHints(rep.Hints)
	rep.Summary = buildSummary(rep.Hints)
	return rep, nil
}

func buildSummary(hints []Hint) string {
	if len(hints) == 0 {
		return "No heuristic hints found; archive looks healthy/empty for the v1 scanners."
	}
	counts := map[Kind]int{}
	for _, h := range hints {
		counts[h.Kind]++
	}
	parts := make([]string, 0, len(counts))
	// Stable order by Kind constants for goldens.
	for _, k := range []Kind{
		KindCrashLoopBackOff,
		KindOOMKilled,
		KindImagePullBackOff,
		KindNotReady,
		KindEvicted,
	} {
		if n := counts[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", k, n))
		}
	}
	return fmt.Sprintf("%d hint(s): %s", len(hints), joinComma(parts))
}

func joinComma(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		out := parts[0]
		for _, p := range parts[1:] {
			out += ", " + p
		}
		return out
	}
}
