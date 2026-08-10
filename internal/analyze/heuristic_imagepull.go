package analyze

import (
	"fmt"
	"strings"
)

// detectImagePullBackOff emits a hint when ImagePullBackOff or ErrImagePull
// with backoff context appears in events or pods JSON waiting reason.
func detectImagePullBackOff(ev evidence) (Hint, bool) {
	if h, ok := imagePullFromPodsJSON(ev); ok {
		return h, true
	}
	if hit := scanEventBodies(ev, imagePullMatch); hit != nil {
		ns, pod := guessPodFromEventLine(hit.line)
		evd := []Evidence{{Path: hit.path, Excerpt: hit.excerpt}}
		evd = enrichCitations(ev, ns, pod, evd)
		return Hint{
			Kind:     KindImagePullBackOff,
			Severity: SeverityError,
			Title:    "Hypothesis: image pull may be in ImagePullBackOff",
			Summary:  "Offline events suggest an image pull backoff (ImagePullBackOff/ErrImagePull). This is a hint, not a definitive registry diagnosis.",
			Evidence: evd,
			OpenQuestions: []string{
				"Is the image name/tag correct and present in the registry?",
				"Do imagePullSecrets or registry credentials need rotation?",
			},
		}, true
	}
	return Hint{}, false
}

func imagePullFromPodsJSON(ev evidence) (Hint, bool) {
	for _, src := range ev.podLists {
		for _, p := range src.List.Items {
			for _, cs := range p.Status.ContainerStatuses {
				reason, msg := waitingReasonMessage(cs)
				if !imagePullReasonOK(reason, msg) {
					continue
				}
				excerpt := clipExcerpt(fmt.Sprintf("pod=%s/%s container=%s waiting.reason=%s",
					podNamespace(p), podName(p), cs.Name, reason))
				evd := []Evidence{{Path: src.Path, Excerpt: excerpt}}
				evd = enrichCitations(ev, podNamespace(p), podName(p), evd)
				return Hint{
					Kind:     KindImagePullBackOff,
					Severity: SeverityError,
					Title:    "Hypothesis: image pull may be in ImagePullBackOff",
					Summary:  "Structured pods JSON shows a waiting reason tied to image pull backoff. This is a hint, not a definitive registry diagnosis.",
					Evidence: evd,
					OpenQuestions: []string{
						"Is the image name/tag correct and present in the registry?",
						"Do imagePullSecrets or registry credentials need rotation?",
					},
				}, true
			}
		}
	}
	return Hint{}, false
}

func waitingReasonMessage(cs containerStatusDTO) (reason, message string) {
	if cs.State.Waiting != nil {
		return cs.State.Waiting.Reason, cs.State.Waiting.Message
	}
	return "", ""
}

func imagePullReasonOK(reason, message string) bool {
	if reason == "ImagePullBackOff" {
		return true
	}
	if reason == "ErrImagePull" {
		// Require backoff context — not generic Failed alone.
		combined := reason + " " + message
		return strings.Contains(combined, "Back-off") ||
			strings.Contains(combined, "Backoff") ||
			strings.Contains(combined, "pulling image") ||
			strings.Contains(message, "ImagePullBackOff")
	}
	return false
}

func imagePullMatch(line string) bool {
	if strings.Contains(line, "ImagePullBackOff") {
		return true
	}
	// ErrImagePull with backoff context on the same line.
	if strings.Contains(line, "ErrImagePull") &&
		(strings.Contains(line, "Back-off") || strings.Contains(line, "Backoff") ||
			strings.Contains(line, "pulling image")) {
		return true
	}
	return false
}
