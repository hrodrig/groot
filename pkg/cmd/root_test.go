package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/groot/pkg/kubetest"
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

func TestRoot_printSampleConfig_goesToStdout(t *testing.T) {
	resetPersistentFlags(t)
	var out, errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"--print-sample-config"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 || !strings.Contains(out.String(), "collection:") {
		t.Fatalf("expected sample on stdout, got %q", out.String())
	}
	if errBuf.Len() != 0 {
		t.Fatalf("sample must not go to stderr (breaks shell redirect); stderr=%q", errBuf.String())
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
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "kubeconfig: " + fmt.Sprintf("%q", filepath.ToSlash(kc)) + "\nnotify:\n  slack: { enabled: false }\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
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
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "kubeconfig: " + fmt.Sprintf("%q", filepath.ToSlash(kc)) + `
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

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestCollect_kubeconfigMissing(t *testing.T) {
	resetPersistentFlags(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(cfgPath, []byte("kubeconfig: \"/no/such/kubeconfig\"\nnotify:\n  slack: { enabled: false }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--config", cfgPath})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error when kubeconfig file is missing")
	}
}

func TestCollect_noNotifySkipsFailedSlack(t *testing.T) {
	resetPersistentFlags(t)
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failSrv.Close)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "kubeconfig: " + fmt.Sprintf("%q", filepath.ToSlash(kc)) + fmt.Sprintf(`
output_dir: %s
notify:
  slack:
    enabled: true
    webhook_url: %q
collection:
  timeout: 30s
  worker_concurrency: 2
  namespaces: []
  include_pod_logs: false
  include_node_details: false
  include_node_logs: false
  include_pod_metrics: false
`, filepath.ToSlash(dir), failSrv.URL)
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--config", cfgPath})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error when slack webhook returns 500")
	}

	resetPersistentFlags(t)
	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"collect", "--quiet", "--no-notify", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("with --no-notify expected success: %v", err)
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
