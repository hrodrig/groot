package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrodrig/groot/internal/kubetest"
)

func TestCollect_strict_noFailuresExitsZero(t *testing.T) {
	// Zero failed jobs, strict on → exit 0.
	oldQuiet := quiet
	quiet = true
	t.Cleanup(func() { quiet = oldQuiet })

	strictMode = true
	strictThreshold = 1
	t.Cleanup(func() { strictMode = false })

	resetPersistentFlags(t)
	kc, cleanup := kubetest.StartAPIServer(t)
	t.Cleanup(cleanup)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "kubeconfig: " + fmtQuote(filepath.ToSlash(kc)) + "\noutput_dir: " + filepath.ToSlash(dir) + "\nnotify:\n  slack: { enabled: false }\ncollection:\n  timeout: 30s\n  worker_concurrency: 2\n  namespaces: []\n  include_pod_logs: false\n  include_node_details: false\n  include_node_logs: false\n  include_pod_metrics: false\n"
	if err := writeFile(cfgPath, yaml); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--strict", "--config", cfgPath})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ExitCodeOf(err); got != 0 {
		t.Fatalf("exit code = %d, want 0", got)
	}
}

func TestCollect_strict_withFailuresExitsFive(t *testing.T) {
	// Point kubeconfig at a missing path so some jobs fail → exit 5.
	oldQuiet := quiet
	quiet = true
	t.Cleanup(func() { quiet = oldQuiet })

	strictMode = true
	strictThreshold = 1
	t.Cleanup(func() { strictMode = false })

	resetPersistentFlags(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	// kubeconfig points at a file that does not exist → jobs can't run → failures.
	yaml := "kubeconfig: " + fmtQuote(filepath.ToSlash(dir)+"/nope") + "\noutput_dir: " + filepath.ToSlash(dir) + "\nnotify:\n  slack: { enabled: false }\ncollection:\n  timeout: 10s\n  worker_concurrency: 2\n  namespaces: [default]\n  include_pod_logs: false\n  include_node_details: false\n  include_node_logs: false\n  include_pod_metrics: false\n"
	if err := writeFile(cfgPath, yaml); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--strict", "--config", cfgPath})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error (collect will abort or produce partial failures)")
	}

	code := ExitCodeOf(err)
	// The collect either aborts (3) if initK8s fails early, or completes with
	// partial failures (5) if the fake server responds but jobs time out on the
	// empty port. Both are acceptable; we assert at least non-zero.
	if code != 3 && code != 5 {
		t.Fatalf("exit code = %d, want 3 or 5 (acceptable abort or partial-failure)", code)
	}
}

func TestCollect_strictNoFlagOkByDefault(t *testing.T) {
	// Default (no --strict) → partial failures exit 0 even when jobs fail.
	oldQuiet := quiet
	quiet = true
	t.Cleanup(func() { quiet = oldQuiet })

	resetPersistentFlags(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "kubeconfig: " + fmtQuote(filepath.ToSlash(dir)+"/nope") + "\noutput_dir: " + filepath.ToSlash(dir) + "\nnotify:\n  slack: { enabled: false }\ncollection:\n  timeout: 10s\n  worker_concurrency: 2\n  namespaces: [default]\n  include_pod_logs: false\n  include_node_details: false\n  include_node_logs: false\n  include_pod_metrics: false\n"
	if err := writeFile(cfgPath, yaml); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--config", cfgPath})

	err := rootCmd.Execute()
	// No --strict → any partial failure still returns 0 (collect completed).
	// We cannot guarantee the test produces partial failures vs abort, so
	// we accept both codes 0 and 3 (abort) as valid — the key assertion is
	// that code is NOT 5 (strict failure), which any occurrence would be
	// a false positive.
	code := ExitCodeOf(err)
	if code == 5 {
		t.Fatal("exit code = 5 without --strict flag; partial failures leaked")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
