package kubetest

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestStartAPIServer_version(t *testing.T) {
	t.Parallel()
	kc, cleanup := StartAPIServer(t)
	t.Cleanup(cleanup)

	resp, err := http.Get(readServerURLFromKubeconfig(t, kc) + "/version")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) < 10 || string(body)[0] != '{' {
		t.Fatalf("unexpected body: %s", body)
	}
}

func readServerURLFromKubeconfig(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// kubeconfigYAML uses "    server: " + srv.URL
	const needle = "server: "
	s := string(data)
	i := strings.Index(s, needle)
	if i < 0 {
		t.Fatal("server not found in kubeconfig")
	}
	rest := s[i+len(needle):]
	j := strings.IndexByte(rest, '\n')
	if j < 0 {
		t.Fatal("newline after server")
	}
	return strings.TrimSpace(rest[:j])
}
