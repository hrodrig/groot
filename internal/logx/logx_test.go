package logx

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_quietSuppressesInfo(t *testing.T) {
	var out, errBuf bytes.Buffer
	l := &Logger{verbose: true, quiet: true, color: false, out: &out, errOut: &errBuf}
	l.Info("x")
	l.Warn("y")
	if out.Len() != 0 {
		t.Fatalf("quiet should suppress info/warn, got %q", out.String())
	}
	l.Error("z")
	if !strings.Contains(errBuf.String(), "ERR") {
		t.Fatalf("errors should print: %q", errBuf.String())
	}
}

func TestLogger_verboseCmdAndOK(t *testing.T) {
	var out, errBuf bytes.Buffer
	l := &Logger{verbose: true, quiet: false, color: false, out: &out, errOut: &errBuf}
	l.Cmd("kubectl %s", "get")
	l.OK("done")
	if !strings.Contains(out.String(), "CMD") || !strings.Contains(out.String(), "OK") {
		t.Fatalf("output: %q", out.String())
	}
}

func TestLogger_noVerboseSkipsCmdOK(t *testing.T) {
	var out bytes.Buffer
	l := &Logger{verbose: false, quiet: false, color: false, out: &out, errOut: &out}
	l.Cmd("x")
	l.OK("y")
	if strings.Contains(out.String(), "CMD") {
		t.Fatalf("unexpected CMD: %q", out.String())
	}
}

func TestLogger_colorAddsANSICodes(t *testing.T) {
	var out bytes.Buffer
	l := &Logger{verbose: false, quiet: false, color: true, out: &out, errOut: &out}
	l.Info("hi")
	s := out.String()
	if !strings.Contains(s, "\033[") {
		t.Fatalf("expected ANSI color in %q", s)
	}
}
