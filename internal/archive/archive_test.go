package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func readTarGzRegularFiles(archivePath string) (map[string][]byte, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	out := make(map[string][]byte)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		out[h.Name] = b
	}
}

func TestDirToTarGz_roundTrip(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(src, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "out.tar.gz")
	if err := DirToTarGz(src, archivePath); err != nil {
		t.Fatalf("DirToTarGz: %v", err)
	}

	files, err := readTarGzRegularFiles(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(src)
	keyA := base + "/a.txt"
	keyB := base + "/nested/b.txt"
	if string(files[keyA]) != "hello" {
		t.Fatalf("%s: %q (keys=%v)", keyA, files[keyA], fileKeys(files))
	}
	if string(files[keyB]) != "world" {
		t.Fatalf("%s: %v", keyB, files)
	}
}

func fileKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestDirToTarGz_createFails(t *testing.T) {
	err := DirToTarGz(t.TempDir(), filepath.Join(t.TempDir(), "nope", "x.tar.gz"))
	if err == nil {
		t.Fatal("expected error creating archive in non-existent parent path")
	}
}
