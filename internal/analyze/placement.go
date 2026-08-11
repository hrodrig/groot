package analyze

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type placementRow struct {
	Namespace string
	Pod       string
	Node      string
	LogFile   string
}

func parsePlacementTSV(body []byte) map[string]placementRow {
	out := map[string]placementRow{}
	if len(body) == 0 {
		return out
	}
	r := csv.NewReader(strings.NewReader(string(body)))
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	nsIdx, podIdx, nodeIdx, logIdx := 0, 1, 2, 3
	headerSeen := false
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) == 0 {
			continue
		}
		if !headerSeen {
			nsIdx, podIdx, nodeIdx, logIdx = parsePlacementHeader(rec)
			headerSeen = true
			if strings.EqualFold(strings.TrimSpace(rec[0]), "namespace") {
				continue
			}
		}
		storePlacementRow(rec, nsIdx, podIdx, nodeIdx, logIdx, out)
	}
	return out
}

func parsePlacementHeader(rec []string) (nsIdx, podIdx, nodeIdx, logIdx int) {
	nsIdx, podIdx, nodeIdx, logIdx = 0, 1, 2, 3
	for i, col := range rec {
		switch strings.TrimSpace(col) {
		case "namespace":
			nsIdx = i
		case "pod":
			podIdx = i
		case "node":
			nodeIdx = i
		case "pod_log_file":
			logIdx = i
		}
	}
	return nsIdx, podIdx, nodeIdx, logIdx
}

func storePlacementRow(rec []string, nsIdx, podIdx, nodeIdx, logIdx int, out map[string]placementRow) {
	ns := fieldAt(rec, nsIdx)
	pod := fieldAt(rec, podIdx)
	if ns == "" || pod == "" {
		return
	}
	out[ns+"/"+pod] = placementRow{
		Namespace: ns,
		Pod:       pod,
		Node:      fieldAt(rec, nodeIdx),
		LogFile:   fieldAt(rec, logIdx),
	}
}

func fieldAt(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

// enrichCitations appends placement/RCA path citations when pod identity is known.
// Never used as a condition detector by itself (ANLZ-10 deferred).
func enrichCitations(ev evidence, ns, pod string, base []Evidence) []Evidence {
	if ns == "" || pod == "" {
		return base
	}
	key := ns + "/" + pod
	row, ok := ev.placementByPod[key]
	if !ok {
		return base
	}
	path := ev.placementPath
	if path == "" {
		path = ev.rcaPath
	}
	if path == "" {
		return base
	}
	excerpt := clipExcerpt(fmt.Sprintf("%s/%s node=%s pod_log_file=%s", ns, pod, row.Node, row.LogFile))
	base = append(base, Evidence{Path: path, Excerpt: excerpt})
	return base
}
