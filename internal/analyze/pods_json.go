package analyze

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Minimal local DTOs for resources.txt "== pods ==" JSON (no client-go).

type podListDTO struct {
	Items []podDTO `json:"items"`
}

type podDTO struct {
	Metadata podMetaDTO   `json:"metadata"`
	Status   podStatusDTO `json:"status"`
}

type podMetaDTO struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type podStatusDTO struct {
	Phase             string               `json:"phase"`
	Reason            string               `json:"reason"`
	Message           string               `json:"message"`
	Conditions        []podConditionDTO    `json:"conditions"`
	ContainerStatuses []containerStatusDTO `json:"containerStatuses"`
}

type podConditionDTO struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type containerStatusDTO struct {
	Name      string            `json:"name"`
	State     containerStateDTO `json:"state"`
	LastState containerStateDTO `json:"lastState"`
}

type containerStateDTO struct {
	Waiting    *containerWaitingDTO    `json:"waiting"`
	Terminated *containerTerminatedDTO `json:"terminated"`
}

type containerWaitingDTO struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type containerTerminatedDTO struct {
	Reason   string `json:"reason"`
	Message  string `json:"message"`
	ExitCode int    `json:"exitCode"`
}

type podListSource struct {
	Path string
	List podListDTO
}

// extractSection returns the body after "== title ==" until the next "== " header.
func extractSection(body []byte, title string) []byte {
	marker := []byte("== " + title + " ==")
	idx := bytes.Index(body, marker)
	if idx < 0 {
		return nil
	}
	rest := body[idx+len(marker):]
	rest = bytes.TrimLeft(rest, "\r\n")
	if end := bytes.Index(rest, []byte("\n== ")); end >= 0 {
		rest = rest[:end]
	}
	return bytes.TrimSpace(rest)
}

func parsePodsSection(body []byte) (podListDTO, error) {
	sec := extractSection(body, "pods")
	if len(sec) == 0 {
		return podListDTO{}, nil
	}
	var list podListDTO
	if err := json.Unmarshal(sec, &list); err != nil {
		return podListDTO{}, err
	}
	return list, nil
}

func podNamespace(p podDTO) string {
	return strings.TrimSpace(p.Metadata.Namespace)
}

func podName(p podDTO) string {
	return strings.TrimSpace(p.Metadata.Name)
}
