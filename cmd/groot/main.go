package main

import (
	"context"
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
	os.Exit(exitCode(run()))
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
