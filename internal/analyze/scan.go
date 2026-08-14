package analyze

import (
	"bufio"
	"bytes"
	"strings"
)

type lineHit struct {
	path    string
	excerpt string
	line    string
}

func scanLines(body []byte, path string, match func(string) bool) *lineHit {
	if len(body) == 0 || path == "" || match == nil {
		return nil
	}
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if match(line) {
			return &lineHit{
				path:    path,
				excerpt: clipExcerpt(strings.TrimSpace(line)),
				line:    line,
			}
		}
	}
	return nil
}

func scanEventBodies(ev evidence, match func(string) bool) *lineHit {
	if h := scanLines(ev.clusterEvents, ev.clusterPath, match); h != nil {
		return h
	}
	if h := scanLines(ev.warningEvents, ev.warningPath, match); h != nil {
		return h
	}
	return scanLines(ev.podsWide, ev.podsWidePath, match)
}
