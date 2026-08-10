package arcread

import (
	"archive/tar"
	"errors"
	"fmt"
	"path/filepath"
)

// Default safety caps for hostile .tar.gz input (D-03).
// Named constants keep production defaults discoverable; tests may use OpenWithCaps with smaller Caps.
const (
	DefaultMaxMemberBytes       int64 = 64 << 20  // 64 MiB = 67108864
	DefaultMaxRegularFiles            = 100_000   // max regular file members per archive
	DefaultMaxDecompressedBytes int64 = 512 << 20 // 512 MiB decompressed total
)

// Caps limits Open Pass-1 indexing and selective reads.
type Caps struct {
	MaxMemberBytes       int64
	MaxRegularFiles      int
	MaxDecompressedBytes int64
}

// DefaultCaps returns the fail-closed production defaults.
func DefaultCaps() Caps {
	return Caps{
		MaxMemberBytes:       DefaultMaxMemberBytes,
		MaxRegularFiles:      DefaultMaxRegularFiles,
		MaxDecompressedBytes: DefaultMaxDecompressedBytes,
	}
}

// Sentinel errors for fail-closed archive reads.
var (
	ErrUnsafePath      = errors.New("arcread: unsafe member path")
	ErrUnsupportedType = errors.New("arcread: unsupported tar type")
	ErrMemberTooLarge  = errors.New("arcread: member exceeds size cap")
	ErrTooManyFiles    = errors.New("arcread: regular file count exceeds cap")
	ErrDecompressedCap = errors.New("arcread: decompressed byte cap exceeded")
	ErrMemberNotFound  = errors.New("arcread: member not found")
	ErrManifestMissing = errors.New("arcread: manifest not found")
	ErrManifestParse   = errors.New("arcread: manifest JSON parse failed")
)

func normalizeMemberName(name string) (string, error) {
	n := filepath.ToSlash(name)
	if n == "" || !filepath.IsLocal(n) {
		return "", fmt.Errorf("%w: %q", ErrUnsafePath, name)
	}
	return n, nil
}

func checkTypeflag(typeflag byte, name string) error {
	switch typeflag {
	case tar.TypeReg, tar.TypeRegA, tar.TypeDir:
		return nil
	case tar.TypeSymlink, tar.TypeLink:
		return fmt.Errorf("%w: %q", ErrUnsupportedType, name)
	default:
		return fmt.Errorf("%w: type %q name %q", ErrUnsupportedType, typeflag, name)
	}
}
