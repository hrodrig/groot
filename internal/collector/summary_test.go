package collector

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestFormatSummary_basicRows(t *testing.T) {
	s := Summary{Total: 10, Success: 9, Failed: 1, Duration: 5 * time.Second, OutputDir: "/out"}
	got := FormatSummary(s, "", UnhealthyPodCounts{})
	wants := []string{"total jobs : 10", "succeeded  : 9", "failed     : 1", "duration   : 5s", "output dir : /out"}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in summary:\n%s", w, got)
		}
	}
	if strings.Contains(got, "archive    :") {
		t.Fatal("archive row should be hidden when empty")
	}
	if strings.Contains(got, "Failures") {
		t.Fatal("failures block should be hidden when empty")
	}
}

func TestFormatSummary_withArchiveAndFailures(t *testing.T) {
	s := Summary{Total: 3, Success: 1, Failed: 2, OutputDir: "/o", Failures: []string{"pod-logs: timeout", "events: client error"}}
	got := FormatSummary(s, "/o/x.tar.gz", UnhealthyPodCounts{})
	if !strings.Contains(got, "archive    : /o/x.tar.gz") {
		t.Fatalf("archive missing:\n%s", got)
	}
	if !strings.Contains(got, "Failures (2):") {
		t.Fatalf("failures header missing:\n%s", got)
	}
	if !strings.Contains(got, "pod-logs: timeout") {
		t.Fatalf("failure line missing:\n%s", got)
	}
	if !strings.Contains(got, "events: client error") {
		t.Fatalf("failure line missing:\n%s", got)
	}
}

func TestFormatSummary_withUnhealthy(t *testing.T) {
	s := Summary{Total: 1, Success: 1}
	u := UnhealthyPodCounts{CrashLoopBackOff: 3, OOMKilled: 1}
	got := FormatSummary(s, "", u)
	if !strings.Contains(got, "Unhealthy pods") {
		t.Fatalf("unhealthy section missing:\n%s", got)
	}
	if !strings.Contains(got, "CrashLoopBackOff : 3") {
		t.Fatalf("clb count missing:\n%s", got)
	}
	if !strings.Contains(got, "OOMKilled        : 1") {
		t.Fatalf("oom count missing:\n%s", got)
	}
	// Zero counters render as "0" so operators can see they were checked.
	for _, line := range []string{"ImagePullBackOff : 0", "Pending          : 0"} {
		if !strings.Contains(got, line) {
			t.Fatalf("missing zero line %q in:\n%s", line, got)
		}
	}
}

func TestUnhealthyPodCounts_HasAny(t *testing.T) {
	if (UnhealthyPodCounts{}).HasAny() {
		t.Fatal("zero struct should not have any")
	}
	if !(UnhealthyPodCounts{CrashLoopBackOff: 1}).HasAny() {
		t.Fatal("any non-zero should have any")
	}
}

func TestUnhealthyPodCounts_SortedKeys(t *testing.T) {
	got := UnhealthyPodCounts{}.SortedKeys()
	want := []string{"CrashLoopBackOff", "ImagePullBackOff", "OOMKilled", "Pending"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got=%v want=%v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("pos %d: got %q want %q", i, got[i], w)
		}
	}
}

func TestWriteSummary_routesToWriter(t *testing.T) {
	var buf bytes.Buffer
	WriteSummary(&buf, Summary{Total: 1, Success: 1}, "", UnhealthyPodCounts{})
	if !strings.Contains(buf.String(), "Summary") {
		t.Fatalf("writer should receive a Summary block, got %q", buf.String())
	}
}
