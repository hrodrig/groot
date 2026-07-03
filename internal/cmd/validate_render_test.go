package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/spf13/cobra"
)

// TestRenderValidate_textRoute exercises the default text path: prints header,
// cluster, output dir, and at least one finding.
func TestRenderValidate_textRoute(t *testing.T) {
	res := collector.PreflightResult{
		OK:        true,
		Cluster:   "kind-local",
		OutputDir: "/tmp/out",
		Findings: []collector.PreflightFinding{
			{Severity: collector.PreflightError, Check: "kubernetes.client", Message: "boom"},
			{Severity: collector.PreflightOK, Check: "disk.free", Message: "ok"},
		},
	}
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	renderValidate(cmd, res, "text")
	out := buf.String()
	for _, want := range []string{"Preflight: OK", "kind-local", "/tmp/out", "kubernetes.client", "disk.free", "[ERROR]", "[OK   ]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

// TestRenderValidate_failVerdict flips OK=false and verifies the verdict text
// switches to FAIL while findings remain visible.
func TestRenderValidate_failVerdict(t *testing.T) {
	res := collector.PreflightResult{OK: false, Findings: []collector.PreflightFinding{
		{Severity: collector.PreflightError, Check: "disk.free", Message: "no space"},
	}}
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	renderValidate(cmd, res, "text")
	if !strings.Contains(buf.String(), "Preflight: FAIL") {
		t.Fatalf("expected FAIL verdict, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "[ERROR] disk.free") {
		t.Fatalf("expected error finding on disk.free, got %q", buf.String())
	}
}

// TestRenderValidate_jsonRoute ensures JSON output is well-formed.
func TestRenderValidate_jsonRoute(t *testing.T) {
	res := collector.PreflightResult{OK: true, OutputDir: "/o"}
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	renderValidate(cmd, res, "json")
	out := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(out, "{") {
		t.Fatalf("expected JSON object, got %q", out)
	}
	for _, want := range []string{`"ok": true`, `"output_dir": "/o"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

// TestRenderInspect_textAndJson cover the inspect render branches, including
// an empty files list and a populated manifest.
func TestRenderInspect_textAndJson(t *testing.T) {
	info := collector.InspectInfo{
		ArchivePath:  "/tmp/a.tar.gz",
		ArchiveSize:  1234,
		FileCount:    1,
		Files:        []string{"extras/manifest.json (10 bytes)"},
		ManifestJSON: `{"groot_version":"0.9.0"}`,
	}
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	renderInspect(cmd, info, "text")
	if !strings.Contains(buf.String(), "Archive: /tmp/a.tar.gz") {
		t.Fatalf("missing archive header: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "extras/manifest.json") {
		t.Fatalf("missing file entry: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "groot_version") {
		t.Fatalf("missing manifest echo: %q", buf.String())
	}

	// Now the JSON route.
	var jbuf bytes.Buffer
	jcmd := &cobra.Command{}
	jcmd.SetOut(&jbuf)
	jcmd.SetErr(&jbuf)
	renderInspect(jcmd, info, "json")
	js := strings.TrimSpace(jbuf.String())
	if !strings.HasPrefix(js, "{") {
		t.Fatalf("expected JSON object, got %q", js)
	}
	if !strings.Contains(js, `"archive_path": "/tmp/a.tar.gz"`) {
		t.Fatalf("missing JSON archive_path: %q", js)
	}
}

func TestSeverityLabel_knownAndUnknown(t *testing.T) {
	cases := []struct {
		in   collector.PreflightSeverity
		want string
	}{
		{collector.PreflightError, "ERROR"},
		{collector.PreflightWarn, "WARN "},
		{collector.PreflightOK, "OK   "},
		{collector.PreflightSeverity("nope"), "????"},
	}
	for _, tc := range cases {
		if got := severityLabel(tc.in); got != tc.want {
			t.Fatalf("severityLabel(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestVerdict(t *testing.T) {
	if verdict(true) != "OK" {
		t.Fatal("ok should be OK")
	}
	if verdict(false) != "FAIL" {
		t.Fatal("false should be FAIL")
	}
}

func TestHumanBytes_routes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{1023, "1023B"},
		{1024, "1.0KiB"},
		{1024 * 1024, "1.0MiB"},
		{1024 * 1024 * 1024, "1.0GiB"},
	}
	for _, tc := range cases {
		got := humanBytes(tc.in)
		if got != tc.want {
			t.Fatalf("humanBytes(%d)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestIndent_preservesNewlinesAndPrefix(t *testing.T) {
	got := indent("a\nb\n", "> ")
	want := "> a\n> b\n"
	if got != want {
		t.Fatalf("indent=%q want %q", got, want)
	}
}
