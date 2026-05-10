package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrodrig/groot/pkg/cmd"
	"github.com/hrodrig/groot/pkg/kubemock"
)

func TestExitCode(t *testing.T) {
	if exitCode(nil) != 0 {
		t.Fatal("nil err => 0")
	}
	if exitCode(errors.New("x")) != 1 {
		t.Fatal("err => 1")
	}
}

func TestRun_printSampleConfig(t *testing.T) {
	old := os.Args
	t.Cleanup(func() {
		os.Args = old
		cmd.ResetPersistentCLI()
	})
	cmd.ResetPersistentCLI()
	os.Args = []string{"groot", "--print-sample-config"}
	if err := run(); err != nil {
		t.Fatal(err)
	}
}

func TestRun_collectQuiet(t *testing.T) {
	cleanupK := kubemock.Install(t)
	defer cleanupK()

	old := os.Args
	t.Cleanup(func() {
		os.Args = old
		cmd.ResetPersistentCLI()
	})
	cmd.ResetPersistentCLI()

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
  include_node_logs: false
  include_pod_metrics: false
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"groot", "collect", "--quiet", "--config", cfgPath}
	if err := run(); err != nil {
		t.Fatal(err)
	}
}
