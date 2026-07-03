package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const redactedToken = "[REDACTED]"

var defaultRedactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|authorization)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*`),
	regexp.MustCompile(`(?i)(aws_secret_access_key|private_key)\s*[:=]\s*\S+`),
}

// RedactCaptureLogs scans collected log files under root and replaces likely secret values.
func RedactCaptureLogs(root string, extraPatterns []string) error {
	patterns := append([]*regexp.Regexp{}, defaultRedactPatterns...)
	for _, raw := range extraPatterns {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		re, err := regexp.Compile(raw)
		if err != nil {
			return fmt.Errorf("invalid redact pattern %q: %w", raw, err)
		}
		patterns = append(patterns, re)
	}

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".log") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		out := string(data)
		for _, re := range patterns {
			out = re.ReplaceAllString(out, redactedToken)
		}
		if out == string(data) {
			return nil
		}
		if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		return nil
	})
}
