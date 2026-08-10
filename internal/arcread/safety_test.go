package arcread_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrodrig/groot/internal/arcread"
)

// writeHostileTarGz builds a .tar.gz fixture from the given headers/bodies.
// Callers must not extract the archive; Open must fail closed in memory.
func writeHostileTarGz(t *testing.T, dest string, entries []hostileEntry) {
	t.Helper()
	f, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)
	for _, e := range entries {
		hdr := e.Header
		if hdr.Size == 0 && len(e.Body) > 0 && (hdr.Typeflag == tar.TypeReg || hdr.Typeflag == 0) {
			hdr.Size = int64(len(e.Body))
		}
		if hdr.Typeflag == 0 {
			hdr.Typeflag = tar.TypeReg
		}
		if hdr.Mode == 0 {
			hdr.Mode = 0o644
		}
		if err := tw.WriteHeader(&hdr); err != nil {
			t.Fatal(err)
		}
		if len(e.Body) > 0 {
			if _, err := tw.Write(e.Body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
}

type hostileEntry struct {
	Header tar.Header
	Body   []byte
}

func TestOpen_HostilePathsFailClosed(t *testing.T) {
	cases := []struct {
		name string
		hdr  tar.Header
		want error
	}{
		{
			name: "parent_segment",
			hdr:  tar.Header{Name: "../evil.txt", Typeflag: tar.TypeReg, Size: 1, Mode: 0o644},
			want: arcread.ErrUnsafePath,
		},
		{
			name: "absolute_unix",
			hdr:  tar.Header{Name: "/etc/passwd", Typeflag: tar.TypeReg, Size: 1, Mode: 0o644},
			want: arcread.ErrUnsafePath,
		},
		{
			name: "nested_dotdot",
			hdr:  tar.Header{Name: "safe/../../escape.txt", Typeflag: tar.TypeReg, Size: 1, Mode: 0o644},
			want: arcread.ErrUnsafePath,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "hostile.tar.gz")
			body := []byte("x")
			tc.hdr.Size = int64(len(body))
			writeHostileTarGz(t, path, []hostileEntry{{Header: tc.hdr, Body: body}})

			_, err := arcread.Open(path)
			if err == nil {
				t.Fatal("Open: want error, got nil")
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("Open: err=%v want errors.Is(..., %v)", err, tc.want)
			}
		})
	}
}

func TestOpen_HostileSymlinkAndHardlinkFailClosed(t *testing.T) {
	cases := []struct {
		name string
		hdr  tar.Header
	}{
		{
			name: "symlink",
			hdr: tar.Header{
				Name:     "link-to-secret",
				Typeflag: tar.TypeSymlink,
				Linkname: "/etc/passwd",
				Mode:     0o777,
			},
		},
		{
			name: "hardlink",
			hdr: tar.Header{
				Name:     "hard-to-secret",
				Typeflag: tar.TypeLink,
				Linkname: "extras/manifest.json",
				Mode:     0o644,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "hostile.tar.gz")
			writeHostileTarGz(t, path, []hostileEntry{{Header: tc.hdr}})

			_, err := arcread.Open(path)
			if err == nil {
				t.Fatal("Open: want error, got nil")
			}
			if !errors.Is(err, arcread.ErrUnsupportedType) {
				t.Fatalf("Open: err=%v want errors.Is(..., ErrUnsupportedType)", err)
			}
		})
	}
}

func TestOpenWithCaps_OversizedMember(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.tar.gz")
	// Small Caps keep the fixture fast; Open rejects when hdr.Size exceeds MaxMemberBytes.
	body := []byte("0123456789abcdef") // 16 bytes
	writeHostileTarGz(t, path, []hostileEntry{{
		Header: tar.Header{
			Name:     "nodes/huge.bin",
			Typeflag: tar.TypeReg,
			Size:     int64(len(body)),
			Mode:     0o644,
		},
		Body: body,
	}})

	_, err := arcread.OpenWithCaps(path, arcread.Caps{
		MaxMemberBytes:       8,
		MaxRegularFiles:      100,
		MaxDecompressedBytes: 1024 * 1024,
	})
	if err == nil {
		t.Fatal("OpenWithCaps: want ErrMemberTooLarge")
	}
	if !errors.Is(err, arcread.ErrMemberTooLarge) {
		t.Fatalf("OpenWithCaps: err=%v want errors.Is(..., ErrMemberTooLarge)", err)
	}
}

func TestOpenWithCaps_TooManyFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "many.tar.gz")
	entries := make([]hostileEntry, 0, 3)
	for i := 0; i < 3; i++ {
		name := filepath.ToSlash(filepath.Join("files", "f"+string(rune('a'+i))+".txt"))
		body := []byte("x")
		entries = append(entries, hostileEntry{
			Header: tar.Header{Name: name, Typeflag: tar.TypeReg, Size: int64(len(body)), Mode: 0o644},
			Body:   body,
		})
	}
	writeHostileTarGz(t, path, entries)

	_, err := arcread.OpenWithCaps(path, arcread.Caps{
		MaxMemberBytes:       1024,
		MaxRegularFiles:      2,
		MaxDecompressedBytes: 1024 * 1024,
	})
	if err == nil {
		t.Fatal("OpenWithCaps: want ErrTooManyFiles")
	}
	if !errors.Is(err, arcread.ErrTooManyFiles) {
		t.Fatalf("OpenWithCaps: err=%v want errors.Is(..., ErrTooManyFiles)", err)
	}
}

func TestOpenWithCaps_DecompressedByteBomb(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bomb.tar.gz")
	body := bytes.Repeat([]byte("Z"), 64)
	entries := []hostileEntry{
		{Header: tar.Header{Name: "a.bin", Typeflag: tar.TypeReg, Size: int64(len(body)), Mode: 0o644}, Body: body},
		{Header: tar.Header{Name: "b.bin", Typeflag: tar.TypeReg, Size: int64(len(body)), Mode: 0o644}, Body: body},
	}
	writeHostileTarGz(t, path, entries)

	_, err := arcread.OpenWithCaps(path, arcread.Caps{
		MaxMemberBytes:       1024,
		MaxRegularFiles:      100,
		MaxDecompressedBytes: 100, // two 64-byte members exceed
	})
	if err == nil {
		t.Fatal("OpenWithCaps: want ErrDecompressedCap")
	}
	if !errors.Is(err, arcread.ErrDecompressedCap) {
		t.Fatalf("OpenWithCaps: err=%v want errors.Is(..., ErrDecompressedCap)", err)
	}
}

func TestOpen_TruncatedGzipAndTarFailClosed(t *testing.T) {
	t.Run("truncated_gzip", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "trunc.gz")
		// Valid gzip magic prefix but truncated payload.
		if err := os.WriteFile(path, []byte{0x1f, 0x8b, 0x08, 0x00}, 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := arcread.Open(path)
		if err == nil {
			t.Fatal("Open: want truncated gzip error")
		}
	})

	t.Run("truncated_tar", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "trunc.tar.gz")
		var buf bytes.Buffer
		gzw := gzip.NewWriter(&buf)
		if _, err := gzw.Write([]byte("not-a-complete-tar-header")); err != nil {
			t.Fatal(err)
		}
		if err := gzw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := arcread.Open(path)
		if err == nil {
			t.Fatal("Open: want truncated tar error")
		}
	})
}

func TestOpen_SafeArchiveDoesNotExtractToDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "safe.tar.gz")
	body := []byte("ok")
	writeHostileTarGz(t, path, []hostileEntry{{
		Header: tar.Header{Name: "extras/note.txt", Typeflag: tar.TypeReg, Size: int64(len(body)), Mode: 0o644},
		Body:   body,
	}})

	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	arc, err := arcread.Open(path)
	if err != nil {
		t.Fatalf("Open safe archive: %v", err)
	}
	defer arc.Close()

	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("Open created extract entries under %s: before=%d after=%d", dir, len(before), len(after))
	}
	for _, e := range after {
		if e.IsDir() {
			t.Fatalf("unexpected extract directory %q under archive workspace", e.Name())
		}
	}
}

func TestDefaultCaps_namedConstants(t *testing.T) {
	if arcread.DefaultMaxMemberBytes != 67108864 {
		t.Fatalf("DefaultMaxMemberBytes=%d want 67108864 (64 MiB)", arcread.DefaultMaxMemberBytes)
	}
	if arcread.DefaultMaxRegularFiles != 100_000 {
		t.Fatalf("DefaultMaxRegularFiles=%d want 100000", arcread.DefaultMaxRegularFiles)
	}
	if arcread.DefaultMaxDecompressedBytes != 512<<20 {
		t.Fatalf("DefaultMaxDecompressedBytes=%d want 512 MiB", arcread.DefaultMaxDecompressedBytes)
	}
	c := arcread.DefaultCaps()
	if c.MaxMemberBytes != arcread.DefaultMaxMemberBytes ||
		c.MaxRegularFiles != arcread.DefaultMaxRegularFiles ||
		c.MaxDecompressedBytes != arcread.DefaultMaxDecompressedBytes {
		t.Fatalf("DefaultCaps mismatch: %+v", c)
	}
}
