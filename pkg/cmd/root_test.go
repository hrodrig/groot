package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groot/pkg/kubemock"
)

func resetPersistentFlags(t *testing.T) {
	t.Helper()
	ResetPersistentCLI()
}

func TestRoot_printSampleConfig(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--print-sample-config"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "groot-capture") {
		t.Fatalf("output: %s", buf.String())
	}
}

func TestCollect_printSampleConfig(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"collect", "--print-sample-config"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "collection:") {
		t.Fatalf("output: %s", buf.String())
	}
}

func TestRoot_testConnection(t *testing.T) {
	resetPersistentFlags(t)
	cleanup := kubemock.Install(t)
	defer cleanup()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(cfgPath, []byte("notify:\n  slack: { enabled: false }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--test-connection", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Kubernetes connection: OK") {
		t.Fatalf("out=%s", buf.String())
	}
}

func TestCollect_quietRun(t *testing.T) {
	resetPersistentFlags(t)
	cleanup := kubemock.Install(t)
	defer cleanup()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := `
output_dir: ` + filepath.ToSlash(dir) + `
notify:
  slack: { enabled: false }
  teams: { enabled: false }
  telegram: { enabled: false }
collection:
  timeout: 30s
  worker_concurrency: 2
  namespaces: []
  include_pod_logs: false
  include_node_details: false
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestCollect_kubectlMissing(t *testing.T) {
	resetPersistentFlags(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(cfgPath, []byte("notify:\n  slack: { enabled: false }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", old) })
	_ = os.Setenv("PATH", t.TempDir())

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--config", cfgPath})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error when kubectl missing")
	}
}

func TestRoot_version(t *testing.T) {
	resetPersistentFlags(t)
	t.Cleanup(func() { SetBuildInfo("dev", "unknown", "unknown", "unknown") })
	SetBuildInfo("0.9.9", "abc", "dev", "2000-01-01")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "0.9.9") {
		t.Fatalf("version output: %s", buf.String())
	}
}

func TestRoot_help(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "groot") {
		t.Fatalf("help: %s", buf.String())
	}
}
