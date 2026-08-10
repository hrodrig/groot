package analyze

import (
	"fmt"
	"strings"
)

// detectEvicted emits a hint for Evicted events or Failed phase with reason Evicted.
func detectEvicted(ev evidence) (Hint, bool) {
	if h, ok := evictedFromPodsJSON(ev); ok {
		return h, true
	}
	if hit := scanEventBodies(ev, evictedMatch); hit != nil {
		ns, pod := guessPodFromEventLine(hit.line)
		evd := []Evidence{{Path: hit.path, Excerpt: hit.excerpt}}
		evd = enrichCitations(ev, ns, pod, evd)
		return Hint{
			Kind:     KindEvicted,
			Severity: SeverityError,
			Title:    "Hypothesis: pod may have been Evicted",
			Summary:  "Offline events mention Evicted. This is a hint based on archive evidence, not a definitive node-pressure diagnosis.",
			Evidence: evd,
			OpenQuestions: []string{
				"Was the node under DiskPressure or MemoryPressure at eviction time?",
				"Are PriorityClass / eviction thresholds appropriate for this workload?",
			},
		}, true
	}
	return Hint{}, false
}

func evictedFromPodsJSON(ev evidence) (Hint, bool) {
	for _, src := range ev.podLists {
		for _, p := range src.List.Items {
			if p.Status.Phase == "Failed" && p.Status.Reason == "Evicted" {
				excerpt := clipExcerpt(fmt.Sprintf("pod=%s/%s phase=Failed reason=Evicted message=%s",
					podNamespace(p), podName(p), p.Status.Message))
				evd := []Evidence{{Path: src.Path, Excerpt: excerpt}}
				evd = enrichCitations(ev, podNamespace(p), podName(p), evd)
				return Hint{
					Kind:     KindEvicted,
					Severity: SeverityError,
					Title:    "Hypothesis: pod may have been Evicted",
					Summary:  "Structured pods JSON shows phase Failed with reason Evicted. This is a hint, not a definitive node-pressure diagnosis.",
					Evidence: evd,
					OpenQuestions: []string{
						"Was the node under DiskPressure or MemoryPressure at eviction time?",
						"Are PriorityClass / eviction thresholds appropriate for this workload?",
					},
				}, true
			}
		}
	}
	return Hint{}, false
}

func evictedMatch(line string) bool {
	// Require Evicted as reason/message token — not DiskPressure alone.
	if !strings.Contains(line, "Evicted") {
		return false
	}
	// Reject lines that only mention eviction thresholds without the Evicted reason word as a field-like hit.
	return true
}
