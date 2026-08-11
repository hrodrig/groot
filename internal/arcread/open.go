package arcread

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Archive is an indexed offline reader for a groot .tar.gz.
type Archive struct {
	path          string
	size          int64
	caps          Caps
	members       []MemberMeta
	manifestCache []byte
	closed        bool
}

// Open opens path with DefaultCaps.
func Open(path string) (*Archive, error) {
	return OpenWithCaps(path, DefaultCaps())
}

// OpenWithCaps indexes the archive under the given caps (fail closed).
func OpenWithCaps(path string, caps Caps) (*Archive, error) {
	caps = normalizeCaps(caps)
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", abs, err)
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", abs, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gzr.Close()

	a := &Archive{
		path: abs,
		size: fi.Size(),
		caps: caps,
	}
	if err := a.indexPass(gzr); err != nil {
		return nil, err
	}
	return a, nil
}

func normalizeCaps(caps Caps) Caps {
	d := DefaultCaps()
	if caps.MaxMemberBytes <= 0 {
		caps.MaxMemberBytes = d.MaxMemberBytes
	}
	if caps.MaxRegularFiles <= 0 {
		caps.MaxRegularFiles = d.MaxRegularFiles
	}
	if caps.MaxDecompressedBytes <= 0 {
		caps.MaxDecompressedBytes = d.MaxDecompressedBytes
	}
	return caps
}

func (a *Archive) indexPass(r io.Reader) error {
	tr := tar.NewReader(r)
	var decompressed int64
	ordinal := -1

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}

		name, skip, err := a.validateHeader(hdr)
		if err != nil {
			return err
		}
		if skip {
			continue
		}

		lr := io.LimitReader(tr, a.caps.MaxMemberBytes+1)
		n, manifestBody, err := drainMember(lr, isManifestName(name))
		if err != nil {
			return fmt.Errorf("drain member %q: %w", name, err)
		}
		if n > a.caps.MaxMemberBytes {
			return fmt.Errorf("%w: %q", ErrMemberTooLarge, name)
		}
		decompressed += n
		if decompressed > a.caps.MaxDecompressedBytes {
			return ErrDecompressedCap
		}

		ordinal++
		a.members = append(a.members, MemberMeta{
			Name:     name,
			Size:     hdr.Size,
			Typeflag: hdr.Typeflag,
			Ordinal:  ordinal,
		})
		if manifestBody != nil && a.manifestCache == nil {
			a.manifestCache = manifestBody
		}
	}
}

func (a *Archive) validateHeader(hdr *tar.Header) (name string, skip bool, err error) {
	name, err = normalizeMemberName(hdr.Name)
	if err != nil {
		return "", false, err
	}
	if err := checkTypeflag(hdr.Typeflag, name); err != nil {
		return "", false, err
	}
	if hdr.Typeflag == tar.TypeDir {
		return name, true, nil
	}
	if hdr.Size > a.caps.MaxMemberBytes {
		return "", false, fmt.Errorf("%w: %q size %d", ErrMemberTooLarge, name, hdr.Size)
	}
	if len(a.members)+1 > a.caps.MaxRegularFiles {
		return "", false, ErrTooManyFiles
	}
	return name, false, nil
}

func drainMember(lr io.Reader, isManifest bool) (n int64, manifestBody []byte, err error) {
	if isManifest {
		manifestBody, err = io.ReadAll(lr)
		if err != nil {
			return 0, nil, err
		}
		return int64(len(manifestBody)), manifestBody, nil
	}
	n, err = io.Copy(io.Discard, lr)
	return n, nil, err
}

// Close releases resources. Safe to call multiple times.
func (a *Archive) Close() error {
	if a == nil {
		return nil
	}
	a.closed = true
	a.manifestCache = nil
	return nil
}

// Path returns the absolute archive path.
func (a *Archive) Path() string {
	if a == nil {
		return ""
	}
	return a.path
}

// Size returns on-disk archive bytes from Open-time Stat.
func (a *Archive) Size() int64 {
	if a == nil {
		return 0
	}
	return a.size
}

func (a *Archive) ensureOpen() error {
	if a == nil || a.closed {
		return fmt.Errorf("arcread: archive is closed")
	}
	return nil
}
