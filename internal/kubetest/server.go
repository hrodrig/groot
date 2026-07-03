// Package kubetest provides a minimal fake Kubernetes API HTTP server for tests.
package kubetest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// StartAPIServer returns a kubeconfig path and cleanup. Covers minimal reads used by groot collect.
func StartAPIServer(t *testing.T) (kubeconfigPath string, cleanup func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(fakeKubeHandler))
	dir := t.TempDir()
	kc := filepath.Join(dir, "config")
	if err := os.WriteFile(kc, []byte(kubeconfigYAML(srv.URL)), 0o600); err != nil {
		srv.Close()
		t.Fatal(err)
	}
	return kc, srv.Close
}

func kubeconfigYAML(serverURL string) string {
	return `apiVersion: v1
kind: Config
current-context: testctx
contexts:
- name: testctx
  context:
    cluster: testcluster
    user: testuser
clusters:
- name: testcluster
  cluster:
    server: ` + serverURL + `
    insecure-skip-tls-verify: true
users:
- name: testuser
  user: {}
`
}
