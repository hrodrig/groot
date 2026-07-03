package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// NormalizePodLogsSince returns a value suitable for kubectl logs --since=..., or empty if raw is empty.
// A string of digits only is treated as a whole number of hours (e.g. "24" -> "24h", "024" -> "24h").
// Otherwise the string must parse with time.ParseDuration (e.g. "45m", "168h", "90s").
func NormalizePodLogsSince(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if isAllASCII_digits(s) {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return "", fmt.Errorf("pod_logs_since: bare number must be a positive hour count, got %q", raw)
		}
		return fmt.Sprintf("%dh", n), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return "", fmt.Errorf("pod_logs_since: invalid value %q: %w (use a duration like 24h, 45m, or a bare hour count like 24)", raw, err)
	}
	if d <= 0 {
		return "", fmt.Errorf("pod_logs_since: duration must be positive, got %q", raw)
	}
	return s, nil
}

func isAllASCII_digits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
