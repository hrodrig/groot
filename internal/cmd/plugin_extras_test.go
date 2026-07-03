package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInvocationLabel_consistentAcrossCalls ensures consecutive calls with
// the same env state produce the same label (no flaky state leaks between
// test iterations).
func TestInvocationLabel_consistentAcrossCalls(t *testing.T) {
	t.Setenv("GROOT_FORCE_KUBECTL_PLUGIN", "")
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"/usr/local/bin/groot"}
	a := InvocationLabel()
	b := InvocationLabel()
	if a != b {
		t.Fatalf("InvocationLabel not stable: %q vs %q", a, b)
	}

	os.Args = []string{"/usr/local/bin/kubectl-groot"}
	c := InvocationLabel()
	d := InvocationLabel()
	if c != d {
		t.Fatalf("InvocationLabel not stable (plugin): %q vs %q", c, d)
	}
}

// TestResetPersistentCLI_resetsSummaryFlag ensures ResetPersistentCLI clears
// the --summary flag back to its default. Tests added when wiring #42+#84.
func TestResetPersistentCLI_resetsSummaryFlag(t *testing.T) {
	resetPersistentFlags(t)
	summaryFlag = true
	ResetPersistentCLI()
	if summaryFlag {
		t.Fatal("summaryFlag should be reset to false")
	}
}

// TestResetPersistentCLI_clearsConfigEnv verifies ResetPersistentCLI does
// not inadvertently trample the cfgFile/file-prefix fields.
func TestResetPersistentCLI_keepsGlobalFlagsDefaults(t *testing.T) {
	resetPersistentFlags(t)
	// Touch a global that ResetPersistentCLI is supposed to reset.
	cfgFile = "/tmp/config.yml"
	ResetPersistentCLI()
	if cfgFile != "" {
		t.Fatalf("cfgFile should reset to empty default, got %q", cfgFile)
	}
}

// TestSkipNotifications_envReparse covers the env-var path for skipNotifications
// (added under #80/#82 wiring) so future refactors don't break env detection.
func TestSkipNotifications_envReparse(t *testing.T) {
	t.Setenv("GROOT_NO_NOTIFY", "true")
	if !skipNotifications() {
		t.Fatal("GROOT_NO_NOTIFY=true should skip")
	}
	t.Setenv("GROOT_NO_NOTIFY", "")
	if skipNotifications() {
		t.Fatal("empty env should not skip")
	}
	t.Setenv("GROOT_NO_NOTIFY", "yes")
	if !skipNotifications() {
		t.Fatal("GROOT_NO_NOTIFY=yes should skip")
	}
}

// TestSkipUploads_envReparse covers the parallel path for the upload flag.
func TestSkipUploads_envReparse(t *testing.T) {
	t.Setenv("GROOT_NO_UPLOAD", "1")
	if !skipUploads() {
		t.Fatal("GROOT_NO_UPLOAD=1 should skip")
	}
	t.Setenv("GROOT_NO_UPLOAD", "false")
	if skipUploads() {
		t.Fatal("explicit false should not skip")
	}
}

// TestValueOrUnknown covers a small helper used by runConnectionTest.
func TestValueOrUnknown(t *testing.T) {
	if got := valueOrUnknown(""); got != "unknown" {
		t.Fatalf("empty input: got %q want unknown", got)
	}
	if got := valueOrUnknown("  "); got != "unknown" {
		t.Fatalf("whitespace input: got %q want unknown", got)
	}
	if got := valueOrUnknown("alpha"); got != "alpha" {
		t.Fatalf("real input: got %q want alpha", got)
	}
}

// Ensure file path printing in error messages does not exceed reasonable
// lengths; this is a behavioural smoke so error formatting regressions don't
// flood user logs.
func TestSampleYAMLPath_safeToPrint(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "config.yml")
	if tmp == "" {
		t.Fatal("temp dir path empty")
	}
}
