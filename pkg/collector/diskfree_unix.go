//go:build !windows

package collector

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
)

// diskFree returns available bytes and total bytes for the filesystem that
// hosts the given path via statfs(2).
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
	free = int64(stat.Bavail) * int64(stat.Bsize)
	total = int64(stat.Blocks) * int64(stat.Bsize)
	return free, total, nil
}
