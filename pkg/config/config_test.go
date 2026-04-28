package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSampleYAML(t *testing.T) {
	s := SampleYAML()
	if !strings.Contains(s, "groot-capture") {
		t.Fatalf("sample should mention file_prefix default: %q", s)
	}
	if !strings.Contains(s, "collection:") {
		t.Fatal("sample should include collection block")
	}
}

func TestLoad_defaultsWhenNoFiles(t *testing.T) {
	root := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)
	t.Setenv("KUBECONFIG", "")
	t.Setenv("GROOT_NOTIFY_SLACK_WEBHOOK_URL", "")
	t.Setenv("GROOT_NOTIFY_TEAMS_WEBHOOK_URL", "")
	t.Setenv("GROOT_NOTIFY_TELEGRAM_TOKEN", "")
	t.Setenv("GROOT_NOTIFY_TELEGRAM_CHAT_ID", "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Collection.WorkerConcurrency < 1 {
		t.Fatalf("worker concurrency: %d", cfg.Collection.WorkerConcurrency)
	}
	if !strings.HasSuffix(cfg.OutputDir, "out") && cfg.OutputDir != "" {
		t.Fatalf("unexpected output_dir: %q", cfg.OutputDir)
	}
}

func TestLoad_validFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	content := `
kubeconfig: ""
output_dir: "` + filepath.Join(dir, "captures") + `"
file_prefix: "pfx"
collection:
  timeout: 1m
  worker_concurrency: 2
  namespaces: ["ns-a"]
  include_pod_logs: false
notify:
  slack:
    enabled: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.FilePrefix != "pfx" {
		t.Fatalf("file_prefix: %q", cfg.FilePrefix)
	}
	if cfg.Collection.WorkerConcurrency != 2 {
		t.Fatalf("worker_concurrency: %d", cfg.Collection.WorkerConcurrency)
	}
	if len(cfg.Collection.Namespaces) != 1 || cfg.Collection.Namespaces[0] != "ns-a" {
		t.Fatalf("namespaces: %#v", cfg.Collection.Namespaces)
	}
	if cfg.Collection.IncludePodLogs {
		t.Fatal("expected include_pod_logs false")
	}
}

func TestLoad_slackEnabledMissingURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  slack:
    enabled: true
    webhook_url: ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_SLACK_WEBHOOK_URL", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when slack enabled without webhook")
	}
	if !strings.Contains(err.Error(), "slack") {
		t.Fatalf("error: %v", err)
	}
}

func TestLoad_podLogTailLinesNegativeCoerced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tail.yaml")
	if err := os.WriteFile(path, []byte(`
collection:
  pod_log_tail_lines: -5
notify:
  slack: { enabled: false }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Collection.PodLogTailLines != 1000 {
		t.Fatalf("expected negative coerced to 1000, got %d", cfg.Collection.PodLogTailLines)
	}
}
