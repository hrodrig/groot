package main

import (
	"os"

	"github.com/hrodrig/groot/pkg/cmd"
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
	return cmd.Execute()
}

func exitCode(err error) int {
	return cmd.ExitCodeOf(err)
}
