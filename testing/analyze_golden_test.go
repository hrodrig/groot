package testing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hrodrig/groot/internal/analyze"
	"github.com/hrodrig/groot/internal/archive"
	"github.com/hrodrig/groot/internal/arcread"
)

const updateGoldenEnv = "UPDATE_GOLDEN"

var goldenFixtures = []struct {
	name   string
	format string
}{
	{"healthy", "executive"},
	{"healthy", "llm"},
	{"crashloop", "executive"},
	{"crashloop", "llm"},
	{"oom", "executive"},
	{"oom", "llm"},
	{"imagepull", "executive"},
	{"imagepull", "llm"},
	{"missing-manifest", "executive"},
	{"missing-manifest", "llm"},
}

// TestAnalyzeGolden locks executive and LLM Markdown output for the committed
// fixture corpus. Run with UPDATE_GOLDEN=1 to refresh expected files.
func TestAnalyzeGolden(t *testing.T) {
	fixturesRoot := filepath.Join("fixtures", "archives")
	for _, tc := range goldenFixtures {
		tc := tc
		t.Run(tc.name+"/"+tc.format, func(t *testing.T) {
			fixtureDir := filepath.Join(fixturesRoot, tc.name)
			sessionDir, err := findSessionDir(fixtureDir)
			if err != nil {
				t.Fatalf("find session dir for %s: %v", tc.name, err)
			}

			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, tc.name+".tar.gz")
			if err := archive.DirToTarGz(sessionDir, archivePath); err != nil {
				t.Fatalf("pack fixture %s: %v", tc.name, err)
			}

			arc, err := arcread.Open(archivePath)
			if err != nil {
				t.Fatalf("open archive %s: %v", archivePath, err)
			}
			t.Cleanup(func() { _ = arc.Close() })

			rep, err := analyze.Run(arc)
			if err != nil {
				t.Fatalf("analyze run: %v", err)
			}

			var out string
			switch tc.format {
			case "executive":
				out, err = analyze.RenderExecutive(rep)
			case "llm":
				out, err = analyze.RenderLLM(rep)
			default:
				t.Fatalf("unknown format %q", tc.format)
			}
			if err != nil {
				t.Fatalf("render %s: %v", tc.format, err)
			}

			// Make golden output stable regardless of the temp archive path.
			out = strings.ReplaceAll(out, archivePath, "<archive>")
			out = normalizeGolden(out)

			goldenPath := filepath.Join(fixtureDir, "expected."+tc.format+".md")
			if os.Getenv(updateGoldenEnv) == "1" {
				if err := os.WriteFile(goldenPath, []byte(out), 0644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				t.Logf("updated golden: %s", goldenPath)
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v", goldenPath, err)
			}
			wantStr := normalizeGolden(string(want))
			if diff := cmp.Diff(wantStr, out); diff != "" {
				t.Fatalf("golden mismatch for %s/%s (-want +got):\n%s", tc.name, tc.format, diff)
			}
		})
	}
}

// TestFixtureCorpusExists sanity-checks that all expected fixture source trees
// are present and can be packed.
func TestFixtureCorpusExists(t *testing.T) {
	for _, name := range []string{"healthy", "crashloop", "oom", "imagepull", "missing-manifest"} {
		t.Run(name, func(t *testing.T) {
			fixtureDir := filepath.Join("fixtures", "archives", name)
			sessionDir, err := findSessionDir(fixtureDir)
			if err != nil {
				t.Fatalf("find session dir: %v", err)
			}
			archivePath := filepath.Join(t.TempDir(), name+".tar.gz")
			if err := archive.DirToTarGz(sessionDir, archivePath); err != nil {
				t.Fatalf("pack fixture: %v", err)
			}
			fi, err := os.Stat(archivePath)
			if err != nil {
				t.Fatalf("stat archive: %v", err)
			}
			if fi.Size() == 0 {
				t.Fatalf("packed archive is empty")
			}
		})
	}
}

func findSessionDir(fixtureDir string) (string, error) {
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "session-") {
			return filepath.Join(fixtureDir, e.Name()), nil
		}
	}
	return "", nil
}

func normalizeGolden(s string) string {
	// Trim trailing whitespace on each line and ensure a single trailing newline.
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	out := strings.Join(lines, "\n")
	out = strings.TrimRight(out, "\n") + "\n"
	return out
}
