package analyze

import (
	"fmt"
	"strings"

	"github.com/hrodrig/groot/internal/arcread"
)

// Analyze-local read caps (stricter than arcread defaults for text scans).
const (
	capEventsBytes  = 2 << 20 // 2 MiB
	capTextBytes    = 2 << 20 // 2 MiB for TSV / resources
	maxExcerptRunes = 512     // display excerpt clip
)

// evidence holds capped archive member payloads for heuristics.
type evidence struct {
	clusterEvents  []byte
	clusterPath    string
	warningEvents  []byte
	warningPath    string
	podsWide       []byte
	podsWidePath   string
	placement      []byte
	placementPath  string
	rca            []byte
	rcaPath        string
	podLists       []podListSource
	placementByPod map[string]placementRow
}

func loadEvidence(arc *arcread.Archive, notes *[]Note) (evidence, error) {
	var ev evidence

	ev.clusterEvents, ev.clusterPath = readOptional(arc, "extras/all-cluster-events.log", capEventsBytes, notes)
	ev.warningEvents, ev.warningPath = readOptional(arc, "extras/warning-events.log", capEventsBytes, notes)
	ev.podsWide, ev.podsWidePath = readOptional(arc, "extras/all-pods-wide.txt", capTextBytes, notes)
	ev.placement, ev.placementPath = readOptional(arc, "extras/all-pod-node-placement.tsv", capTextBytes, notes)
	ev.rca, ev.rcaPath = readOptional(arc, "extras/all-pods-rca.tsv", capTextBytes, notes)

	// Prefer placement; fall back to RCA columns for the same join keys.
	ev.placementByPod = parsePlacementTSV(ev.placement)
	if len(ev.placementByPod) == 0 {
		ev.placementByPod = parsePlacementTSV(ev.rca)
	}

	ev.podLists = loadPodLists(arc, notes)

	if len(ev.clusterEvents) == 0 && len(ev.warningEvents) == 0 {
		*notes = append(*notes, Note{
			Code:    "insufficient_evidence",
			Message: "No cluster events members available for heuristic scans",
			Path:    "extras/all-cluster-events.log",
		})
	}
	return ev, nil
}

func loadPodLists(arc *arcread.Archive, notes *[]Note) []podListSource {
	var out []podListSource
	found := false
	for _, m := range arc.Members() {
		if !strings.HasSuffix(m.Name, "/resources.txt") && m.Name != "resources.txt" {
			continue
		}
		found = true
		body, err := arc.ReadMemberLimit(m.Name, capTextBytes)
		if err != nil {
			*notes = append(*notes, Note{
				Code:    "member_read_error",
				Message: fmt.Sprintf("failed to read %q: %v", m.Name, err),
				Path:    m.Name,
			})
			continue
		}
		list, err := parsePodsSection(body)
		if err != nil {
			*notes = append(*notes, Note{
				Code:    "parse_degraded",
				Message: fmt.Sprintf("pods JSON section in %q could not be parsed: %v", m.Name, err),
				Path:    m.Name,
			})
			continue
		}
		if len(list.Items) == 0 {
			continue
		}
		out = append(out, podListSource{Path: m.Name, List: list})
	}
	if !found {
		*notes = append(*notes, Note{
			Code:    "member_missing",
			Message: `optional member matching "*/resources.txt" not found`,
			Path:    "resources.txt",
		})
	}
	return out
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
