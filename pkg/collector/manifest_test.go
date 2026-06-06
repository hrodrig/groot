package collector

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestListCaptureRelPaths(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"extras/a.txt", "kube-system/b.log"} {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := listCaptureRelPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"extras/a.txt", "kube-system/b.log"}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
