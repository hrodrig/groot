package analyze

import (
	"bufio"
	"bytes"
	"strings"
)

// detectCrashLoop emits a CrashLoopBackOff hint when events (or pods-wide text)
// contain waiting/backoff evidence. Requires a cited archive member path.
func detectCrashLoop(ev evidence) (Hint, bool) {
	type hit struct {
		path    string
		excerpt string
	}
	var found *hit

	scan := func(body []byte, path string) {
		if found != nil || len(body) == 0 || path == "" {
			return
		}
		sc := bufio.NewScanner(bytes.NewReader(body))
		// Allow long event lines without failing the scan.
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if crashLoopMatch(line) {
				found = &hit{path: path, excerpt: clipExcerpt(strings.TrimSpace(line))}
				return
			}
		}
	}

	scan(ev.clusterEvents, ev.clusterPath)
	scan(ev.warningEvents, ev.warningPath)
	scan(ev.podsWide, ev.podsWidePath)

	if found == nil {
		return Hint{}, false
	}

	return Hint{
		Kind:     KindCrashLoopBackOff,
		Severity: SeverityError,
		Title:    "Hypothesis: container may be in CrashLoopBackOff",
		Summary:  "Offline events suggest a container restart backoff (CrashLoopBackOff). This is a hint based on archive evidence, not a definitive diagnosis.",
		Evidence: []Evidence{{
			Path:    found.path,
			Excerpt: found.excerpt,
		}},
		OpenQuestions: []string{
			"Which container image or configuration change preceded the restarts?",
			"Do recent logs show an application panic or probe failure before the backoff?",
		},
	}, true
}

func crashLoopMatch(line string) bool {
	if strings.Contains(line, "CrashLoopBackOff") {
		return true
	}
	// Common kubelet event message tied to CrashLoopBackOff.
	if strings.Contains(line, "Back-off restarting failed container") {
		return true
	}
	return false
}
