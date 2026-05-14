package kubeloader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRESTConfig_missingExplicitFile(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "does-not-exist")
	_, err := RESTConfig(p)
	if err == nil || !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("expected kubeconfig error, got %v", err)
	}
}

func TestAPIConfig_missingExplicitFile(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "missing-kubeconfig")
	_, err := APIConfig(p)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRESTConfig_and_APIConfig_minimalFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	kc := filepath.Join(dir, "config")
	content := `apiVersion: v1
kind: Config
current-context: c1
contexts:
- name: c1
  context: {cluster: cl, user: u}
clusters:
- name: cl
  cluster: {server: https://127.0.0.1:6443}
users:
- name: u
  user: {}
`
	if err := os.WriteFile(kc, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := RESTConfig(kc)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "https://127.0.0.1:6443" {
		t.Fatalf("host %q", cfg.Host)
	}
	raw, err := APIConfig(kc)
	if err != nil {
		t.Fatal(err)
	}
	if raw.CurrentContext != "c1" {
		t.Fatalf("context %q", raw.CurrentContext)
	}
}
