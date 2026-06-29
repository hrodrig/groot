package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// These tests assert the documented exit code taxonomy from plan-0.9.0 § #82
// (SPEC §3 exit semantics) for representative failure paths through the CLI.
// They never require a real cluster — bad config and missing kubeconfig are
// enough to exercise config and kubernetes code paths.

func TestCLI_exitCode_missingKubeconfig_isCollectAborted(t *testing.T) {
	resetPersistentFlags(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(cfgPath, []byte("kubeconfig: \"/no/such/kubeconfig\"\nnotify:\n  slack: { enabled: false }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--config", cfgPath})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error from missing kubeconfig")
	}
	// config.Load accepts the missing path today (it stores the path and lets
	// the runtime client fail). That pushes the failure into collector.Run →
	// kubeloader.RESTConfig → so the wrapper is ExitCollectAborted (the
	// Kubernetes-API failure happened during collect execution). The
	// kubeconfig path itself is a config-tier concern — distinguishing the two
	// would require pushing the kubeconfig existence check earlier into config
	// loading, tracked separately.
	if got := ExitCodeOf(err); got != ExitCollectAborted {
		t.Fatalf("exit code = %d, want %d (collect aborted)", got, ExitCollectAborted)
	}
}

func TestCLI_exitCode_badSinceFlag_isConfigCode(t *testing.T) {
	resetPersistentFlags(t)
	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--since", "not-a-duration"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error from invalid --since")
	}
	if got := ExitCodeOf(err); got != ExitConfigError {
		t.Fatalf("exit code = %d, want %d (config)", got, ExitConfigError)
	}
}

func TestCLI_exitCode_missingConfigFile_isConfigCode(t *testing.T) {
	resetPersistentFlags(t)
	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	nonexistent := filepath.Join(t.TempDir(), "does-not-exist.yml")
	rootCmd.SetArgs([]string{"collect", "--quiet", "--config", nonexistent})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error from missing config file")
	}
	if got := ExitCodeOf(err); got != ExitConfigError {
		t.Fatalf("exit code = %d, want %d (config)", got, ExitConfigError)
	}
}
