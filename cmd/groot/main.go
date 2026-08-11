// Package main is the groot CLI entrypoint. Logic lives in internal/cmd.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hrodrig/groot/internal/cmd"
)

var (
	version   = "dev"
	commit    = "unknown"
	branch    = "unknown"
	buildDate = "unknown"
)

func main() {
	os.Exit(runMain())
}

// runMain runs the CLI and returns a process exit code (see cmd.ExitCodeOf).
func runMain() int {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return exitCode(err)
	}
	return 0
}

func run() error {
	cmd.SetBuildInfo(version, commit, branch, buildDate)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return cmd.ExecuteContext(ctx)
}

func exitCode(err error) int {
	return cmd.ExitCodeOf(err)
}
