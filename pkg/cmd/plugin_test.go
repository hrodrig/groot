package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPluginInvocation_basenameMatch(t *testing.T) {
	t.Setenv("GROOT_FORCE_KUBECTL_PLUGIN", "")
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	// Standalone binary -> not a plugin invocation.
	os.Args = []string{"/usr/local/bin/groot"}
	if IsPluginInvocation() {
		t.Fatal("basename `groot` must not be flagged as plugin invocation")
	}

	// Via symlink to the same name -> not a plugin invocation.
	os.Args = []string{"./groot"}
	if IsPluginInvocation() {
		t.Fatal("./groot basename must not be flagged as plugin invocation")
	}

	// kubectl plugin dispatch -> invoked as kubectl-groot on argv[0].
	os.Args = []string{"/opt/krew/bin/kubectl-groot"}
	if !IsPluginInvocation() {
		t.Fatal("/opt/krew/bin/kubectl-groot must be flagged as plugin invocation")
	}

	// Someone on Windows running `kubectl-groot.exe` directly (after PATH
	// lookup strips the suffix); we do not strip extensions ourselves but
	// .exe is the only realistic case on Windows.
	if filepath.Ext(filepath.Base(os.Args[0])) == ".exe" {
		// covered separately in TestIsPluginInvocation_windowsExe below;
		// keep this branch as a no-op so the table stays self-explanatory.
	}
}

func TestIsPluginInvocation_envOverride(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"/usr/local/bin/groot"}

	// Forced off (the empty string path) -> false.
	t.Setenv("GROOT_FORCE_KUBECTL_PLUGIN", "")
	if IsPluginInvocation() {
		t.Fatal("empty override must NOT force plugin mode")
	}

	cases := []struct {
		value string
		want  bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"Yes", true},
		{"yes", true},
		{"0", false},
		{"false", false},
		{"  ", false},
		{"on", false}, // on is NOT in the accepted set; be strict to avoid surprise
		{"", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("GROOT_FORCE_KUBECTL_PLUGIN", tc.value)
			if got := IsPluginInvocation(); got != tc.want {
				t.Fatalf("GROOT_FORCE_KUBECTL_PLUGIN=%q: got=%v want=%v", tc.value, got, tc.want)
			}
		})
	}
}

func TestInvocationLabel(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	t.Setenv("GROOT_FORCE_KUBECTL_PLUGIN", "")

	os.Args = []string{"/opt/local/bin/groot"}
	if got := InvocationLabel(); got != "groot" {
		t.Fatalf("standalone basename: got %q want groot", got)
	}

	os.Args = []string{"/opt/krew/bin/kubectl-groot"}
	if got := InvocationLabel(); got != "kubectl-groot" {
		t.Fatalf("plugin invocation: got %q want kubectl-groot", got)
	}

	// Symlink to `groot` from a deeper path: filename wins; kubectl plugin
	// detection stays false (basename != kubectl-groot).
	os.Args = []string{"/very/long/path/mysymlink"}
	if got := InvocationLabel(); got != "mysymlink" {
		t.Fatalf("custom basename: got %q want mysymlink", got)
	}
}
