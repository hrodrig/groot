package analyze

import (
	"fmt"

	"github.com/hrodrig/groot/internal/arcread"
)

// Analyze-local read caps (stricter than arcread defaults for text scans).
const (
	capEventsBytes  = 2 << 20 // 2 MiB
	capTextBytes    = 2 << 20 // 2 MiB for TSV / resources
	maxExcerptRunes = 512     // display excerpt clip; pod-log ReadMemberLimit (4 KiB) arrives with 02-02
)

// evidence holds capped archive member payloads for heuristics.
type evidence struct {
	clusterEvents []byte
	clusterPath   string
	warningEvents []byte
	warningPath   string
	podsWide      []byte
	podsWidePath  string
}

func loadEvidence(arc *arcread.Archive, notes *[]Note) (evidence, error) {
	var ev evidence

	ev.clusterEvents, ev.clusterPath = readOptional(arc, "extras/all-cluster-events.log", capEventsBytes, notes)
	ev.warningEvents, ev.warningPath = readOptional(arc, "extras/warning-events.log", capEventsBytes, notes)
	ev.podsWide, ev.podsWidePath = readOptional(arc, "extras/all-pods-wide.txt", capTextBytes, notes)

	if len(ev.clusterEvents) == 0 && len(ev.warningEvents) == 0 {
		*notes = append(*notes, Note{
			Code:    "insufficient_evidence",
			Message: "No cluster events members available for heuristic scans",
			Path:    "extras/all-cluster-events.log",
		})
	}
	return ev, nil
}

func readOptional(arc *arcread.Archive, suffix string, limit int64, notes *[]Note) ([]byte, string) {
	meta, ok := arc.LookupSuffix(suffix)
	if !ok {
		*notes = append(*notes, Note{
			Code:    "member_missing",
			Message: fmt.Sprintf("optional member %q not found", suffix),
			Path:    suffix,
		})
		return nil, ""
	}
	body, err := arc.ReadMemberLimit(meta.Name, limit)
	if err != nil {
		*notes = append(*notes, Note{
			Code:    "member_read_error",
			Message: fmt.Sprintf("failed to read %q: %v", meta.Name, err),
			Path:    meta.Name,
		})
		return nil, meta.Name
	}
	return body, meta.Name
}

func clipExcerpt(s string) string {
	r := []rune(s)
	if len(r) <= maxExcerptRunes {
		return s
	}
	return string(r[:maxExcerptRunes]) + "…"
}
