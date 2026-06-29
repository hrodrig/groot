package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletion_bash(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"completion", "bash"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "bash") && !strings.Contains(out, "_groot") && !strings.Contains(out, "complete") {
		t.Fatalf("bash completion must reference shell completion primitives, got: %s", trim(out))
	}
}

func TestCompletion_zsh(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"completion", "zsh"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "compdef") && !strings.Contains(out, "groot") {
		t.Fatalf("zsh completion must define compdef or reference groot, got: %s", trim(out))
	}
}

func TestCompletion_fish(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"completion", "fish"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "complete") && !strings.Contains(out, "groot") {
		t.Fatalf("fish completion must reference `complete`, got: %s", trim(out))
	}
}

func TestCompletion_powershell(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"completion", "powershell"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "groot") && !strings.Contains(out, "Register-ArgumentCompleter") {
		t.Fatalf("powershell completion must reference groot or Register-ArgumentCompleter, got: %s", trim(out))
	}
}

func TestCompletion_rejectsUnknownShell(t *testing.T) {
	resetPersistentFlags(t)
	var out, errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"completion", "tcsh"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("error should mention unsupported shell, got: %v", err)
	}
}

func TestCompletion_rejectsZeroArgs(t *testing.T) {
	resetPersistentFlags(t)
	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"completion"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing shell argument")
	}
}

func TestCompletion_rejectsTooManyArgs(t *testing.T) {
	resetPersistentFlags(t)
	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"completion", "bash", "zsh"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for too many shell arguments")
	}
}

func TestCompletion_listedInRoot(t *testing.T) {
	resetPersistentFlags(t)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "completion") {
		t.Fatalf("root help should list the completion command, got: %s", buf.String())
	}
}

func trim(s string) string {
	if len(s) > 400 {
		return s[:400] + "... (truncated)"
	}
	return s
}
