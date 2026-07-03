//go:build openbsd

package collector

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
)

// diskFree returns available bytes and total bytes for the filesystem that
// hosts the given path via statfs(2). OpenBSD Statfs_t uses F_* field names.
func diskFree(path string) (free, total int64, err error) {
	if strings.TrimSpace(path) == "" {
		return 0, 0, errors.New("output_dir is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve path %q: %w", path, err)
	}
	var stat syscall.Statfs_t
	if serr := syscall.Statfs(abs, &stat); serr != nil {
		return 0, 0, fmt.Errorf("stat %s: %w", abs, serr)
	}
	free = stat.F_bavail * int64(stat.F_bsize)
	total = int64(stat.F_blocks) * int64(stat.F_bsize)
	return free, total, nil
}
