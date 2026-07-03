package collector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactCaptureLogs_replacesSecrets(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "default", "pod.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "line1\npassword=supersecret\nBearer abc.def.ghi\n"
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RedactCaptureLogs(dir, nil); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "supersecret") || strings.Contains(s, "abc.def.ghi") {
		t.Fatalf("secrets not redacted: %q", s)
	}
	if !strings.Contains(s, redactedToken) {
		t.Fatalf("expected %q in %q", redactedToken, s)
	}
}

func TestRedactCaptureLogs_skipsNonLogs(t *testing.T) {
	dir := t.TempDir()
	txt := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(txt, []byte("password=keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RedactCaptureLogs(dir, nil); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(txt)
	if string(out) != "password=keep" {
		t.Fatalf("non-log file changed: %q", out)
	}
}

func TestRedactCaptureLogs_invalidPattern(t *testing.T) {
	dir := t.TempDir()
	err := RedactCaptureLogs(dir, []string{"("})
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}
