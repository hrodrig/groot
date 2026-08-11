package analyze_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/analyze"
	"github.com/hrodrig/groot/internal/archive"
	"github.com/hrodrig/groot/internal/arcread"
)

func packFixture(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("testdata", name)
	out := filepath.Join(t.TempDir(), name+".tar.gz")
	if err := archive.DirToTarGz(src, out); err != nil {
		t.Fatalf("DirToTarGz: %v", err)
	}
	return out
}

func TestRun_CrashLoopBackOff(t *testing.T) {
	path := packFixture(t, "crashloop")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.RunID != "run-crashloop-001" {
		t.Fatalf("RunID = %q", rep.RunID)
	}
	if rep.ArchiveSHA256 != "abc123deadbeef0123456789abcdef" {
		t.Fatalf("ArchiveSHA256 = %q", rep.ArchiveSHA256)
	}
	if len(rep.Hints) != 1 {
		t.Fatalf("hints = %d, want 1: %+v", len(rep.Hints), rep.Hints)
	}
	h := rep.Hints[0]
	if h.Kind != analyze.KindCrashLoopBackOff {
		t.Fatalf("kind = %q", h.Kind)
	}
	if h.Severity != analyze.SeverityError {
		t.Fatalf("severity = %v", h.Severity)
	}
	if len(h.Evidence) == 0 || h.Evidence[0].Path == "" {
		t.Fatalf("missing evidence path: %+v", h.Evidence)
	}
	if !strings.Contains(h.Evidence[0].Path, "all-cluster-events.log") {
		t.Fatalf("evidence path = %q", h.Evidence[0].Path)
	}
	if !strings.Contains(h.Evidence[0].Excerpt, "CrashLoopBackOff") {
		t.Fatalf("excerpt missing CrashLoopBackOff: %q", h.Evidence[0].Excerpt)
	}
	if strings.Contains(strings.ToLower(h.Summary), "root cause") {
		t.Fatalf("summary must not claim root cause: %q", h.Summary)
	}
}

func TestRun_HealthyEmpty(t *testing.T) {
	path := packFixture(t, "healthy")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rep.Hints) != 0 {
		t.Fatalf("expected zero hints, got %+v", rep.Hints)
	}
	if !strings.Contains(strings.ToLower(rep.Summary), "healthy") &&
		!strings.Contains(strings.ToLower(rep.Summary), "empty") {
		t.Fatalf("summary should mention healthy/empty: %q", rep.Summary)
	}
	if rep.RunID != "run-healthy-001" {
		t.Fatalf("RunID = %q", rep.RunID)
	}
}

func TestRun_ReportJSONRoundTrip(t *testing.T) {
	path := packFixture(t, "crashloop")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var got analyze.Report
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, b)
	}
	if got.RunID != rep.RunID || got.ArchiveSHA256 != rep.ArchiveSHA256 {
		t.Fatalf("header mismatch: %+v", got)
	}
	if len(got.Hints) != 1 || got.Hints[0].Kind != analyze.KindCrashLoopBackOff {
		t.Fatalf("hints mismatch: %+v", got.Hints)
	}
	if got.Hints[0].Severity != analyze.SeverityError {
		t.Fatalf("severity JSON round-trip failed: %v", got.Hints[0].Severity)
	}
	if !strings.Contains(string(b), `"severity": "error"`) {
		t.Fatalf("severity must marshal as string error:\n%s", b)
	}
}
