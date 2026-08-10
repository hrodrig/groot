package analyze

import (
	"fmt"
	"strings"
)

// detectOOMKilled emits an OOMKilled hint only when evidence contains the
// OOMKilled reason text. Exit code 137 alone is never sufficient (pitfall guard).
func detectOOMKilled(ev evidence, notes *[]Note) (Hint, bool) {
	if h, ok := oomFromPodsJSON(ev); ok {
		return h, true
	}
	if hit := scanEventBodies(ev, oomKilledMatch); hit != nil {
		ns, pod := guessPodFromEventLine(hit.line)
		evd := []Evidence{{Path: hit.path, Excerpt: hit.excerpt}}
		evd = enrichCitations(ev, ns, pod, evd)
		return Hint{
			Kind:     KindOOMKilled,
			Severity: SeverityError,
			Title:    "Hypothesis: container may have been OOMKilled",
			Summary:  "Offline evidence mentions OOMKilled. This is a hint based on archive text, not a definitive memory diagnosis.",
			Evidence: evd,
			OpenQuestions: []string{
				"Was the container memory limit too low for the workload peak?",
				"Do recent logs show allocation spikes before termination?",
			},
		}, true
	}

	// Soft signal: exit 137 without OOMKilled — note only, never invent OOM hint.
	if notes != nil && exit137WithoutOOM(ev) {
		*notes = append(*notes, Note{
			Code:    "open_question",
			Message: "Exit code 137 observed without OOMKilled reason text; not treated as OOMKilled (may be SIGKILL for other causes)",
			Path:    firstAvailablePath(ev),
		})
	}
	return Hint{}, false
}

func oomFromPodsJSON(ev evidence) (Hint, bool) {
	for _, src := range ev.podLists {
		for _, p := range src.List.Items {
			for _, cs := range p.Status.ContainerStatuses {
				if reason := terminatedReason(cs); reason == "OOMKilled" {
					excerpt := clipExcerpt(fmt.Sprintf("pod=%s/%s container=%s terminated.reason=OOMKilled",
						podNamespace(p), podName(p), cs.Name))
					evd := []Evidence{{Path: src.Path, Excerpt: excerpt}}
					evd = enrichCitations(ev, podNamespace(p), podName(p), evd)
					return Hint{
						Kind:     KindOOMKilled,
						Severity: SeverityError,
						Title:    "Hypothesis: container may have been OOMKilled",
						Summary:  "Structured pods JSON shows terminated reason OOMKilled. This is a hint, not a definitive root-cause diagnosis.",
						Evidence: evd,
						OpenQuestions: []string{
							"Was the container memory limit too low for the workload peak?",
							"Do recent logs show allocation spikes before termination?",
						},
					}, true
				}
			}
		}
	}
	return Hint{}, false
}

func terminatedReason(cs containerStatusDTO) string {
	if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
		return cs.State.Terminated.Reason
	}
	if cs.LastState.Terminated != nil && cs.LastState.Terminated.Reason != "" {
		return cs.LastState.Terminated.Reason
	}
	return ""
}

func oomKilledMatch(line string) bool {
	return strings.Contains(line, "OOMKilled")
}

func exit137WithoutOOM(ev evidence) bool {
	saw137 := false
	check := func(body []byte) {
		if len(body) == 0 {
			return
		}
		s := string(body)
		if strings.Contains(s, "OOMKilled") {
			return
		}
		if strings.Contains(s, "exit code 137") || strings.Contains(s, "ExitCode:137") ||
			strings.Contains(s, `"exitCode": 137`) || strings.Contains(s, `"exitCode":137`) {
			saw137 = true
		}
	}
	check(ev.clusterEvents)
	check(ev.warningEvents)
	check(ev.podsWide)
	for _, src := range ev.podLists {
		for _, p := range src.List.Items {
			for _, cs := range p.Status.ContainerStatuses {
				if strings.Contains(terminatedReason(cs), "OOMKilled") {
					return false
				}
				ec := 0
				if cs.State.Terminated != nil {
					ec = cs.State.Terminated.ExitCode
				} else if cs.LastState.Terminated != nil {
					ec = cs.LastState.Terminated.ExitCode
				}
				if ec == 137 {
					saw137 = true
				}
			}
		}
	}
	return saw137
}

func firstAvailablePath(ev evidence) string {
	for _, p := range []string{ev.clusterPath, ev.warningPath, ev.podsWidePath} {
		if p != "" {
			return p
		}
	}
	for _, src := range ev.podLists {
		if src.Path != "" {
			return src.Path
		}
	}
	return "extras/all-cluster-events.log"
}

// guessPodFromEventLine best-effort extracts namespace/pod from kubectl event OBJECT column.
func guessPodFromEventLine(line string) (ns, pod string) {
	// Common shapes: "pod/demo-0" or "default/demo-0"
	fields := strings.Fields(line)
	for _, f := range fields {
		if strings.HasPrefix(f, "pod/") {
			pod = strings.TrimPrefix(f, "pod/")
			return "", pod
		}
		if i := strings.Index(f, "/"); i > 0 && !strings.Contains(f, ":") {
			// skip timestamps / urls
			left, right := f[:i], f[i+1:]
			if left != "" && right != "" && !strings.ContainsAny(left, ".") {
				return left, right
			}
		}
	}
	return "", ""
}
