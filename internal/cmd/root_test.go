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

	"github.com/hrodrig/groot/internal/kubetest"
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

func TestCollect_listJobs(t *testing.T) {
	resetPersistentFlags(t)
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "kubeconfig: " + fmt.Sprintf("%q", filepath.ToSlash(kc)) + `
collection:
  timeout: 30s
  worker_concurrency: 1
  namespaces: []
  include_pod_logs: false
  include_node_details: false
  include_node_logs: false
  include_pod_metrics: false
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"collect", "--list-jobs", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "cluster-info") {
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
	for _, args := range [][]string{{"--version"}, {"-v"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			resetPersistentFlags(t)
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(args)
			err := Execute()
			if err != nil {
				t.Fatal(err)
			}
			if got := ExitCodeOf(err); got != ExitSuccess {
				t.Fatalf("exit code = %d, want %d", got, ExitSuccess)
			}
			out := buf.String()
			if !strings.Contains(out, "groot v0.9.9") {
				t.Fatalf("version output: %s", out)
			}
			if strings.Contains(out, "I am Groot") {
				t.Fatalf("short version should not include greeting: %s", out)
			}
			// Regression: PersistentPreRunE sentinel must not dump cobra Usage/Error (kzero-style Silence*).
			if strings.Contains(out, "Usage:") || strings.Contains(out, "Error:") {
				t.Fatalf("version must not print Usage/Error: %q", out)
			}
			lines := strings.Split(strings.TrimSpace(out), "\n")
			if len(lines) != 1 {
				t.Fatalf("want single version line, got %d: %q", len(lines), out)
			}
		})
	}
}

func TestRoot_version_long(t *testing.T) {
	resetPersistentFlags(t)
	t.Cleanup(func() { SetBuildInfo("dev", "unknown", "unknown", "unknown") })
	SetBuildInfo("0.9.9", "abc", "dev", "2000-01-01")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--version", "--long"})
	if err := Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(out), "I am Groot v0.9.9") {
		t.Fatalf("long version output: %s", out)
	}
}

func TestRoot_version_subcommand(t *testing.T) {
	resetPersistentFlags(t)
	t.Cleanup(func() { SetBuildInfo("dev", "unknown", "unknown", "unknown") })
	SetBuildInfo("0.9.9", "abc", "dev", "2000-01-01")
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"version", "--long"})
	if err := Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "I am Groot v0.9.9") {
		t.Fatalf("version subcommand output: %s", buf.String())
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

func TestSkipUploads_flagAndEnv(t *testing.T) {
	resetPersistentFlags(t)
	noUpload = true
	if !skipUploads() {
		t.Fatal("flag")
	}
	noUpload = false
	t.Setenv("GROOT_NO_UPLOAD", "yes")
	if !skipUploads() {
		t.Fatal("env")
	}
}
