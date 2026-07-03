package logx

import (
	"fmt"
	"io"
	"os"
)

const (
	reset  = "\033[0m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	blue   = "\033[34m"
)

// Logger prints structured console messages.
type Logger struct {
	verbose bool
	quiet   bool
	color   bool
	out     io.Writer
	errOut  io.Writer
}

// New creates a logger.
func New(verbose, quiet, noColor bool) *Logger {
	return &Logger{
		verbose: verbose,
		quiet:   quiet,
		color:   !noColor,
		out:     os.Stdout,
		errOut:  os.Stderr,
	}
}

// Info prints informational messages.
func (l *Logger) Info(format string, args ...any) {
	if l.quiet {
		return
	}
	l.print(l.out, "INFO", blue, format, args...)
}

// Warn prints warning messages.
func (l *Logger) Warn(format string, args ...any) {
	if l.quiet {
		return
	}
	l.print(l.out, "WARN", yellow, format, args...)
}

// Error prints error messages.
func (l *Logger) Error(format string, args ...any) {
	l.print(l.errOut, "ERR", red, format, args...)
}

// Cmd prints a command to be executed.
func (l *Logger) Cmd(format string, args ...any) {
	if l.quiet || !l.verbose {
		return
	}
	l.print(l.out, "CMD", cyan, format, args...)
}

// OK prints success messages for completed commands.
func (l *Logger) OK(format string, args ...any) {
	if l.quiet || !l.verbose {
		return
	}
	l.print(l.out, "OK", green, format, args...)
}

func (l *Logger) print(w io.Writer, label, colorCode, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if l.color {
		fmt.Fprintf(w, "%s[%s]%s %s\n", colorCode, label, reset, msg)
		return
	}
	fmt.Fprintf(w, "[%s] %s\n", label, msg)
}
