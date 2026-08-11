package analyze

import "sort"

// sortHints orders by severity rank (error first), then Kind ascending,
// then first evidence path for deterministic goldens (D-06).
func sortHints(hints []Hint) {
	sort.SliceStable(hints, func(i, j int) bool {
		ri, rj := hints[i].Severity.Rank(), hints[j].Severity.Rank()
		if ri != rj {
			return ri < rj
		}
		if hints[i].Kind != hints[j].Kind {
			return hints[i].Kind < hints[j].Kind
		}
		return firstEvidencePath(hints[i]) < firstEvidencePath(hints[j])
	})
}

func firstEvidencePath(h Hint) string {
	if len(h.Evidence) == 0 {
		return ""
	}
	return h.Evidence[0].Path
}
