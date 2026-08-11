package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/archive"
)

func packAnalyzeFixture(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("..", "analyze", "testdata", name)
	out := filepath.Join(t.TempDir(), name+".tar.gz")
	if err := archive.DirToTarGz(src, out); err != nil {
		t.Fatalf("DirToTarGz: %v", err)
	}
	return out
}

func TestAnalyze_CrashLoopExecutiveMarkdown(t *testing.T) {
	resetPersistentFlags(t)
	arcPath := packAnalyzeFixture(t, "crashloop")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"analyze", arcPath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("analyze: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "CrashLoopBackOff") {
		t.Fatalf("stdout missing CrashLoopBackOff:\n%s", out)
	}
	if !strings.Contains(out, "# groot analyze") {
		t.Fatalf("stdout missing title:\n%s", out)
	}
	if !strings.Contains(out, "run-crashloop-001") {
		t.Fatalf("stdout missing run_id:\n%s", out)
	}
}

func TestAnalyze_JSONOutput(t *testing.T) {
	resetPersistentFlags(t)
	arcPath := packAnalyzeFixture(t, "crashloop")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"analyze", "--output", "json", arcPath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("analyze json: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, buf.String())
	}
	if got["run_id"] != "run-crashloop-001" {
		t.Fatalf("run_id = %v", got["run_id"])
	}
	if got["archive_sha256"] != "abc123deadbeef0123456789abcdef" {
		t.Fatalf("archive_sha256 = %v", got["archive_sha256"])
	}
	hints, ok := got["hints"].([]any)
	if !ok || len(hints) != 1 {
		t.Fatalf("hints = %#v", got["hints"])
	}
	h0, _ := hints[0].(map[string]any)
	if h0["kind"] != "CrashLoopBackOff" {
		t.Fatalf("hint kind = %#v", h0["kind"])
	}
	if h0["severity"] != "error" {
		t.Fatalf("severity = %#v", h0["severity"])
	}
	if _, ok := got["summary"].(string); !ok || got["summary"] == "" {
		t.Fatalf("missing summary: %#v", got["summary"])
	}
}

func TestAnalyze_HealthyExitZero(t *testing.T) {
	resetPersistentFlags(t)
	arcPath := packAnalyzeFixture(t, "healthy")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"analyze", arcPath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("healthy analyze should exit 0: %v", err)
	}
	out := strings.ToLower(buf.String())
	if !strings.Contains(out, "healthy") && !strings.Contains(out, "empty") {
		t.Fatalf("healthy summary missing:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "CrashLoopBackOff") {
		t.Fatalf("healthy output must not include CrashLoopBackOff")
	}
}

func TestAnalyze_ExitCollectAbortedOnMissingArchive(t *testing.T) {
	resetPersistentFlags(t)
	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"analyze", "/no/such/archive.tar.gz"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if got := ExitCodeOf(err); got != ExitCollectAborted {
		t.Fatalf("exit code = %d, want %d", got, ExitCollectAborted)
	}
	if !strings.Contains(err.Error(), "analyze archive") {
		t.Fatalf("error must wrap analyze archive: %v", err)
	}
}

func TestAnalyze_ExitCollectAbortedOnCorruptArchive(t *testing.T) {
	resetPersistentFlags(t)
	bad := filepath.Join(t.TempDir(), "broken.tar.gz")
	if err := os.WriteFile(bad, []byte("not gzip"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"analyze", bad})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if got := ExitCodeOf(err); got != ExitCollectAborted {
		t.Fatalf("exit code = %d, want %d", got, ExitCollectAborted)
	}
}

func TestAnalyze_InvalidOutputRejected(t *testing.T) {
	resetPersistentFlags(t)
	arcPath := packAnalyzeFixture(t, "healthy")

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"analyze", "--output", "yaml", arcPath})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected unsupported output error")
	}
	if got := ExitCodeOf(err); got == ExitCollectAborted {
		t.Fatalf("invalid --output must not map to exit 3, got %d", got)
	}
	if !strings.Contains(err.Error(), "unsupported --output") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "llm") {
		t.Fatalf("unsupported --output error should mention llm: %v", err)
	}
}

func TestAnalyze_LLMOutput(t *testing.T) {
	resetPersistentFlags(t)
	arcPath := packAnalyzeFixture(t, "crashloop")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"analyze", "--output", "llm", arcPath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("analyze llm: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Instructions for the assistant",
		"Secrets warning",
		"unredacted",
		"CrashLoopBackOff",
		"generated by groot analyze --output llm",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestAnalyze_HelpMentionsExit3(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"analyze", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "3") || !strings.Contains(strings.ToLower(out), "archive") {
		t.Fatalf("analyze help should document exit 3 for archive failure:\n%s", out)
	}
	if !strings.Contains(out, "text") || !strings.Contains(out, "json") || !strings.Contains(out, "llm") {
		t.Fatalf("analyze help should list text|json|llm:\n%s", out)
	}
}
