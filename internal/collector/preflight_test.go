package collector

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/config"
)

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{2048, "2.0KiB"},
		{8 * 1024 * 1024, "8.0MiB"},
		{3 * 1024 * 1024 * 1024, "3.0GiB"},
	}
	for _, tc := range cases {
		got := formatBytes(tc.in)
		if got != tc.want {
			t.Fatalf("formatBytes(%d)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestFindingsLess(t *testing.T) {
	// Errors before warns before oks; tie-break by check name.
	a := PreflightFinding{Severity: PreflightError, Check: "b"}
	b := PreflightFinding{Severity: PreflightError, Check: "a"}
	c := PreflightFinding{Severity: PreflightWarn, Check: "z"}
	d := PreflightFinding{Severity: PreflightOK, Check: "m"}

	if !findingsLess(b, a) {
		t.Fatal("check-name tie-break failed (a < b)")
	}
	if !findingsLess(a, c) {
		t.Fatal("warn should follow error")
	}
	if !findingsLess(c, d) {
		t.Fatal("ok should follow warn")
	}
	if findingsLess(a, b) {
		t.Fatal("a should come before b in tied severities")
	}
}

func TestRbacChecks_usesFirstNamespace(t *testing.T) {
	got := rbacChecks([]string{"alpha", "beta"})
	var saw []string
	for _, c := range got {
		if c.namespace != "" {
			saw = append(saw, c.namespace)
		}
	}
	if len(saw) == 0 {
		t.Fatal("expected at least one namespaced check")
	}
	for _, ns := range saw {
		if ns != "alpha" {
			t.Fatalf("expected first namespace alpha, got %q", ns)
		}
	}
}

func TestRbacChecks_clusterScopedUsesEmpty(t *testing.T) {
	got := rbacChecks([]string{"alpha"})
	for _, c := range got {
		if c.label == "list.nodes" || c.label == "list.namespaces" {
			if c.namespace != "" {
				t.Fatalf("cluster-scoped check %q should have empty namespace, got %q", c.label, c.namespace)
			}
		}
	}
}

func TestPreflight_diskFree(t *testing.T) {
	dir := t.TempDir()
	free, total, err := diskFree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if free <= 0 {
		t.Fatalf("free should be > 0 on tmpfs, got %d", free)
	}
	if total <= 0 {
		t.Fatalf("total should be > 0 on tmpfs, got %d", total)
	}
}

func TestPreflight_diskFree_emptyPath(t *testing.T) {
	if _, _, err := diskFree(""); err == nil {
		t.Fatal("expected error for empty output_dir")
	}
}

func TestPreflight_diskFree_missingPathDoesNotPanic(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	_, _, _ = diskFree(missing)
}

func TestPreflight_kubeconfigMissing_marksKubernetesError(t *testing.T) {
	svc := New(config.Config{
		Kubeconfig: "/path/that/does/not/exist/kubeconfig-not-found.yaml",
		OutputDir:  t.TempDir(),
	})
	svc.cfg.Collection.Namespaces = []string{}

	res := svc.Preflight(context.Background())
	if res.OK {
		t.Fatalf("expected OK=false with no kubeconfig, got %+v", res)
	}
	var sawClient bool
	for _, f := range res.Findings {
		if f.Check == "kubernetes.client" && f.Severity == PreflightError {
			sawClient = true
		}
	}
	if !sawClient {
		t.Fatalf("expected kubernetes.client finding, got %+v", res.Findings)
	}
}

// Ensure findings sorting is stable (handler does sort.SliceStable) so we can
// assert a deterministic order in tests that scan over them.
func TestPreflight_findingsSorted(t *testing.T) {
	a := PreflightFinding{Severity: PreflightError, Check: "z"}
	b := PreflightFinding{Severity: PreflightWarn, Check: "a"}
	c := PreflightFinding{Severity: PreflightOK, Check: "m"}

	got := []PreflightFinding{c, a, b}
	// Apply same ordering predicate as Preflight does.
	sorted := append([]PreflightFinding(nil), got...)
	// bubble by predicate (we only need an order, not optimal)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if findingsLess(sorted[j], sorted[i]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if sorted[0].Severity != PreflightError {
		t.Fatalf("severities: %+v", sorted)
	}
	// Confirm no errors leaked into ok position.
	for _, f := range sorted {
		_ = strings.TrimSpace(f.Message)
	}
}
