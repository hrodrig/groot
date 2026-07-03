package cmd

import (
	"strings"
	"testing"
)

func TestFormatVersion_short(t *testing.T) {
	t.Cleanup(func() { SetBuildInfo("dev", "unknown", "unknown", "unknown") })
	SetBuildInfo("0.9.9", "abc", "main", "2000-01-01")

	got := FormatVersion(false)
	want := "groot v0.9.9 (commit=abc branch=main built=2000-01-01)"
	if got != want {
		t.Fatalf("FormatVersion(false) = %q, want %q", got, want)
	}
}

func TestFormatVersion_long(t *testing.T) {
	t.Cleanup(func() { SetBuildInfo("dev", "unknown", "unknown", "unknown") })
	SetBuildInfo("0.9.9", "abc", "main", "2000-01-01")

	got := FormatVersion(true)
	if !strings.HasPrefix(got, "I am Groot v0.9.9") {
		t.Fatalf("FormatVersion(true) = %q, want I am Groot prefix", got)
	}
	if !strings.Contains(got, "commit=abc") {
		t.Fatalf("FormatVersion(true) = %q, want commit metadata", got)
	}
}

func TestFormatVersion_short_asPlugin(t *testing.T) {
	t.Cleanup(func() { SetBuildInfo("dev", "unknown", "unknown", "unknown") })
	SetBuildInfo("0.9.9", "abc", "main", "2000-01-01")
	t.Setenv("GROOT_FORCE_KUBECTL_PLUGIN", "1")

	got := FormatVersion(false)
	if !strings.HasPrefix(got, "kubectl-groot v0.9.9") {
		t.Fatalf("FormatVersion(false) under plugin = %q, want kubectl-groot prefix", got)
	}
}

func TestFormatVersion_long_asPluginKeepsGreeting(t *testing.T) {
	t.Cleanup(func() { SetBuildInfo("dev", "unknown", "unknown", "unknown") })
	SetBuildInfo("0.9.9", "abc", "main", "2000-01-01")
	t.Setenv("GROOT_FORCE_KUBECTL_PLUGIN", "true")

	got := FormatVersion(true)
	if !strings.HasPrefix(got, "I am Groot v0.9.9") {
		t.Fatalf("FormatVersion(true) under plugin = %q, want Groot greeting (not kubectl-groot)", got)
	}
}
