package analyze

import (
	"fmt"
	"strings"
)

// detectNotReady prefers structured pods JSON Ready==False over bare substrings.
func detectNotReady(ev evidence) (Hint, bool) {
	if h, ok := notReadyFromPodsJSON(ev); ok {
		return h, true
	}
	// Secondary: readiness probe / NotReady event lines (still require Ready context).
	if hit := scanEventBodies(ev, notReadyEventMatch); hit != nil {
		ns, pod := guessPodFromEventLine(hit.line)
		evd := []Evidence{{Path: hit.path, Excerpt: hit.excerpt}}
		evd = enrichCitations(ev, ns, pod, evd)
		return Hint{
			Kind:     KindNotReady,
			Severity: SeverityWarn,
			Title:    "Hypothesis: workload may be NotReady",
			Summary:  "Offline events suggest a readiness problem. This is a hint based on archive evidence, not a definitive probe diagnosis.",
			Evidence: evd,
			OpenQuestions: []string{
				"Are readiness probes failing, or is the pod still initializing?",
				"Did a recent deploy change probe paths or ports?",
			},
		}, true
	}
	return Hint{}, false
}

func notReadyFromPodsJSON(ev evidence) (Hint, bool) {
	for _, src := range ev.podLists {
		for _, p := range src.List.Items {
			for _, c := range p.Status.Conditions {
				if c.Type == "Ready" && strings.EqualFold(c.Status, "False") {
					excerpt := clipExcerpt(fmt.Sprintf("pod=%s/%s condition Ready=False reason=%s message=%s",
						podNamespace(p), podName(p), c.Reason, c.Message))
					evd := []Evidence{{Path: src.Path, Excerpt: excerpt}}
					evd = enrichCitations(ev, podNamespace(p), podName(p), evd)
					return Hint{
						Kind:     KindNotReady,
						Severity: SeverityWarn,
						Title:    "Hypothesis: pod Ready condition is False",
						Summary:  "Structured pods JSON shows Ready=False. This is a hint, not a definitive readiness-root-cause diagnosis.",
						Evidence: evd,
						OpenQuestions: []string{
							"Are readiness probes failing, or is the pod still initializing?",
							"Did a recent deploy change probe paths or ports?",
						},
					}, true
				}
			}
		}
	}
	return Hint{}, false
}

func notReadyEventMatch(line string) bool {
	// Avoid bare "NotReady" false positives from unrelated text; require readiness signal.
	lower := strings.ToLower(line)
	if strings.Contains(line, "Readiness probe failed") {
		return true
	}
	if strings.Contains(line, "Ready") && strings.Contains(lower, "false") {
		return true
	}
	return false
}
