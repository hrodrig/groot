package collector

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// newRunID generates a per-run identifier (ROADMAP #81) with shape
//
//	YYYYMMDDTHHMMSSZ-<short>
//
// where <short> is the first 4 bytes of crypto/rand encoded as Crockford base32
// (no padding, lowercase-with-uppercase). Examples:
//
//	20260628T153045Z-7KQV
//	20260628T153045Z-9P3W
//
// The timestamp makes the id human-sortable; the suffix keeps it unique even
// across two collects firing in the same second. We deliberately keep the
// suffix short (5 chars) — collisions within a single user's run history are
// astronomically unlikely, and the manifest records the precise wall-clock
// timestamp anyway.
func newRunID() string {
	now := time.Now().UTC()
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Fall back to a deterministic-but-unique suffix drawn from the
		// timestamp nanoseconds. Still unique within a single process.
		b := now.UnixNano()
		buf[0] = byte(b >> 24)
		buf[1] = byte(b >> 16)
		buf[2] = byte(b >> 8)
		buf[3] = byte(b)
	}
	suffix := strings.ToUpper(strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf[:]), "="))
	return fmt.Sprintf("%s-%s", now.Format("20060102T150405Z"), suffix)
}

// fileSHA256 returns the SHA-256 hex digest of the file at the given path.
// Uses streaming reads so the memory footprint stays flat for large archives.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
