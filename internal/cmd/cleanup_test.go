package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// packageOutDir is the default capture root when cmd tests run with output_dir: ./out
// from the internal/cmd working directory (gitignored under internal/cmd/out/).
const packageOutDir = "out"

func TestMain(m *testing.M) {
	code := m.Run()
	cleanupPackageCaptureDirs()
	os.Exit(code)
}

func cleanupPackageCaptureDirs() {
	entries, err := os.ReadDir(packageOutDir)
	if err != nil {
		return
	}
	for _, ent := range entries {
		if !ent.IsDir() || !strings.HasPrefix(ent.Name(), "groot-capture-") {
			continue
		}
		_ = os.RemoveAll(filepath.Join(packageOutDir, ent.Name()))
	}
}
