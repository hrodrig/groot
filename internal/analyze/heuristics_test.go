package analyze_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/analyze"
	"github.com/hrodrig/groot/internal/arcread"
)

func TestRun_OOMKilled(t *testing.T) {
	path := packFixture(t, "oom")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	h := findHint(t, rep, analyze.KindOOMKilled)
	if h.Severity != analyze.SeverityError {
		t.Fatalf("severity = %v", h.Severity)
	}
	if len(h.Evidence) == 0 || h.Evidence[0].Path == "" {
		t.Fatalf("missing evidence: %+v", h.Evidence)
	}
	if !strings.Contains(h.Evidence[0].Excerpt, "OOMKilled") &&
		!evidenceContains(h, "OOMKilled") {
		t.Fatalf("evidence must cite OOMKilled: %+v", h.Evidence)
	}
	// Placement enrichment optional but expected for this fixture.
	if !evidencePathContains(h, "placement") && !evidencePathContains(h, "resources.txt") &&
		!evidencePathContains(h, "all-cluster-events") {
		t.Fatalf("unexpected evidence paths: %+v", h.Evidence)
	}
	if strings.Contains(strings.ToLower(h.Summary), "root cause") {
		t.Fatalf("summary must not claim root cause: %q", h.Summary)
	}
}

func TestRun_Exit137_NoOOMKilled(t *testing.T) {
	path := packFixture(t, "exit137")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, h := range rep.Hints {
		if h.Kind == analyze.KindOOMKilled {
			t.Fatalf("exit 137 alone must not emit OOMKilled hint: %+v", h)
		}
	}
	foundNote := false
	for _, n := range rep.Notes {
		if n.Code == "open_question" && strings.Contains(n.Message, "137") {
			foundNote = true
			break
		}
	}
	if !foundNote {
		t.Fatalf("expected open_question note about exit 137, notes=%+v", rep.Notes)
	}
}

func TestRun_ImagePullBackOff(t *testing.T) {
	path := packFixture(t, "imagepull")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	h := findHint(t, rep, analyze.KindImagePullBackOff)
	if h.Severity != analyze.SeverityError {
		t.Fatalf("severity = %v", h.Severity)
	}
	if len(h.Evidence) == 0 {
		t.Fatal("expected non-empty Evidence")
	}
	if !evidenceContains(h, "ImagePullBackOff") {
		t.Fatalf("evidence must cite ImagePullBackOff: %+v", h.Evidence)
	}
}

func TestRun_NotReady(t *testing.T) {
	path := packFixture(t, "notready")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	h := findHint(t, rep, analyze.KindNotReady)
	if h.Severity != analyze.SeverityWarn {
		t.Fatalf("severity = %v, want warn", h.Severity)
	}
	if !evidencePathContains(h, "resources.txt") {
		t.Fatalf("NotReady should cite resources.txt: %+v", h.Evidence)
	}
	if !evidenceContains(h, "Ready=False") && !evidenceContains(h, "Ready") {
		t.Fatalf("excerpt should cite Ready condition: %+v", h.Evidence)
	}
}

func TestRun_Evicted(t *testing.T) {
	path := packFixture(t, "evicted")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	h := findHint(t, rep, analyze.KindEvicted)
	if len(h.Evidence) == 0 {
		t.Fatal("expected evidence")
	}
	if !evidenceContains(h, "Evicted") {
		t.Fatalf("evidence must cite Evicted: %+v", h.Evidence)
	}
}

func TestRun_MissingExtras_Degrade(t *testing.T) {
	path := packFixture(t, "missing-extras")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v (must degrade, not fail)", err)
	}
	if len(rep.Hints) != 0 {
		t.Fatalf("expected zero invented hints, got %+v", rep.Hints)
	}
	if len(rep.Notes) == 0 {
		t.Fatal("expected member_missing / insufficient_evidence notes")
	}
	codes := map[string]bool{}
	for _, n := range rep.Notes {
		codes[n.Code] = true
	}
	if !codes["member_missing"] && !codes["insufficient_evidence"] {
		t.Fatalf("expected member_missing or insufficient_evidence, got %+v", rep.Notes)
	}
}

func TestRun_Mixed_SortSeverityThenKind(t *testing.T) {
	path := packFixture(t, "mixed")
	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer arc.Close()

	rep, err := analyze.Run(arc)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rep.Hints) < 2 {
		t.Fatalf("want >=2 hints for mixed fixture, got %+v", rep.Hints)
	}
	// error before warn
	sawWarn := false
	for i, h := range rep.Hints {
		if h.Severity == analyze.SeverityWarn {
			sawWarn = true
			continue
		}
		if h.Severity == analyze.SeverityError && sawWarn {
			t.Fatalf("hint[%d] error after warn: %+v", i, rep.Hints)
		}
		if i > 0 {
			prev := rep.Hints[i-1]
			if prev.Severity.Rank() > h.Severity.Rank() {
				t.Fatalf("severity rank out of order at %d: %+v", i, rep.Hints)
			}
			if prev.Severity == h.Severity && prev.Kind > h.Kind {
				t.Fatalf("kind not ascending within severity at %d: %+v", i, rep.Hints)
			}
		}
	}
	if rep.Hints[0].Kind != analyze.KindCrashLoopBackOff {
		t.Fatalf("first hint want CrashLoopBackOff, got %q", rep.Hints[0].Kind)
	}
	if rep.Hints[0].Severity != analyze.SeverityError {
		t.Fatalf("first severity want error")
	}
	findHint(t, rep, analyze.KindNotReady)
}

func TestRun_AllFiveKindsCovered(t *testing.T) {
	fixtures := []struct {
		name string
		kind analyze.Kind
	}{
		{"crashloop", analyze.KindCrashLoopBackOff},
		{"oom", analyze.KindOOMKilled},
		{"imagepull", analyze.KindImagePullBackOff},
		{"notready", analyze.KindNotReady},
		{"evicted", analyze.KindEvicted},
	}
	seen := map[analyze.Kind]bool{}
	for _, tc := range fixtures {
		path := packFixture(t, tc.name)
		arc, err := arcread.Open(path)
		if err != nil {
			t.Fatalf("%s Open: %v", tc.name, err)
		}
		rep, err := analyze.Run(arc)
		_ = arc.Close()
		if err != nil {
			t.Fatalf("%s Run: %v", tc.name, err)
		}
		findHint(t, rep, tc.kind)
		seen[tc.kind] = true
	}
	for _, k := range []analyze.Kind{
		analyze.KindCrashLoopBackOff,
		analyze.KindOOMKilled,
		analyze.KindImagePullBackOff,
		analyze.KindNotReady,
		analyze.KindEvicted,
	} {
		if !seen[k] {
			t.Fatalf("kind %s not exercised", k)
		}
	}
}

func TestImportBoundary_NoClientGo(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`, "./internal/analyze")
	// Run from module root (two levels up from this package's testdata cwd is not reliable).
	// go test sets cwd to the package directory; use module-relative via go list from here.
	cmd.Dir = "../.." // repo root from internal/analyze
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "k8s.io/client-go") {
		t.Fatalf("internal/analyze must not import client-go:\n%s", out)
	}
	// Also check transitive imports of the package itself (Imports is direct only).
	cmd2 := exec.Command("go", "list", "-f", `{{join .Deps "\n"}}`, "./internal/analyze")
	cmd2.Dir = "../.."
	out2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("go list deps: %v\n%s", err, out2)
	}
	for _, line := range strings.Split(string(out2), "\n") {
		if strings.HasPrefix(line, "k8s.io/client-go") {
			t.Fatalf("internal/analyze dependency graph includes client-go: %s", line)
		}
	}
}

func findHint(t *testing.T, rep analyze.Report, kind analyze.Kind) analyze.Hint {
	t.Helper()
	for _, h := range rep.Hints {
		if h.Kind == kind {
			return h
		}
	}
	t.Fatalf("hint kind %s not found in %+v", kind, rep.Hints)
	return analyze.Hint{}
}

func evidenceContains(h analyze.Hint, substr string) bool {
	for _, e := range h.Evidence {
		if strings.Contains(e.Excerpt, substr) || strings.Contains(e.Path, substr) {
			return true
		}
	}
	return false
}

func evidencePathContains(h analyze.Hint, substr string) bool {
	for _, e := range h.Evidence {
		if strings.Contains(e.Path, substr) {
			return true
		}
	}
	return false
}
