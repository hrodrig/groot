package arcread

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// ReadMember returns member bytes capped at MaxMemberBytes.
func (a *Archive) ReadMember(name string) ([]byte, error) {
	if a == nil {
		return nil, ErrMemberNotFound
	}
	return a.ReadMemberLimit(name, a.caps.MaxMemberBytes)
}

// ReadMemberLimit reopens the archive and reads the named member up to limit bytes.
// gzip.Reader is not seekable; this uses ordinal skip (two-pass reopen).
func (a *Archive) ReadMemberLimit(name string, limit int64) ([]byte, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	meta, ok := a.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrMemberNotFound, name)
	}
	if limit <= 0 {
		limit = a.caps.MaxMemberBytes
	}
	if limit > a.caps.MaxMemberBytes {
		limit = a.caps.MaxMemberBytes
	}

	f, err := os.Open(a.path)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	ordinal := -1
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read: %w", err)
		}
		switch hdr.Typeflag {
		case tar.TypeReg:
			ordinal++
		default:
			continue
		}
		if ordinal != meta.Ordinal {
			continue
		}
		lr := io.LimitReader(tr, limit+1)
		buf, err := io.ReadAll(lr)
		if err != nil {
			return nil, fmt.Errorf("read member %q: %w", name, err)
		}
		if int64(len(buf)) > limit {
			return nil, fmt.Errorf("%w: %q", ErrMemberTooLarge, name)
		}
		return buf, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrMemberNotFound, name)
}
