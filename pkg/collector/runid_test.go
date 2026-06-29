package collector

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestNewRunID_shape(t *testing.T) {
	id := newRunID()
	// YYYYMMDDTHHMMSSZ-<chars>
	re := regexp.MustCompile(`^\d{8}T\d{6}Z-[A-Z2-7]{7}$`)
	if !re.MatchString(id) {
		t.Fatalf("unexpected run id shape: %q", id)
	}
}

func TestNewRunID_unique(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 1000; i++ {
		id := newRunID()
		if _, dup := seen[id]; dup {
			t.Fatalf("collision: %q after %d iterations", id, i)
		}
		seen[id] = struct{}{}
	}
}

func TestFileSHA256_matchesStdlib(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blob.txt")
	body := []byte("hello, groot\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := fileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(body)
	if got != hex.EncodeToString(want[:]) {
		t.Fatalf("mismatch: got=%s want=%s", got, hex.EncodeToString(want[:]))
	}
}

func TestFileSHA256_missingPath(t *testing.T) {
	if _, err := fileSHA256(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
