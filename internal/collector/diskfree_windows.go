//go:build windows

package collector

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// diskFree returns available bytes and total bytes for the volume hosting path
// via GetDiskFreeSpaceEx (Windows equivalent of statfs preflight).
func diskFree(path string) (free, total int64, err error) {
	if strings.TrimSpace(path) == "" {
		return 0, 0, errors.New("output_dir is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve path %q: %w", path, err)
	}
	p, err := syscall.UTF16PtrFromString(abs)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve path %q: %w", path, err)
	}
	var freeAvail, totalBytes, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeAvail, &totalBytes, &totalFree); err != nil {
		return 0, 0, fmt.Errorf("stat %s: %w", abs, err)
	}
	_ = totalFree // caller uses total capacity only
	return int64(freeAvail), int64(totalBytes), nil
}
