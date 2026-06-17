package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// ErrVersionPrinted stops command execution after --version output; Execute treats it as success.
type versionPrintedError struct{}

func (versionPrintedError) Error() string { return "" }

// ErrVersionPrinted is returned from PersistentPreRun when --version was handled.
var ErrVersionPrinted = versionPrintedError{}

func initVersionFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolP("version", "v", false, "Print version information and exit")
	cmd.PersistentFlags().BoolP("long", "l", false, "With --version, print the long greeting (I am Groot …)")
	_ = cmd.PersistentFlags().MarkHidden("long")
}

// FormatVersion returns the short line for scripts or the Groot greeting when long is true.
func FormatVersion(long bool) string {
	line := fmt.Sprintf("groot v%s (commit=%s branch=%s built=%s)",
		buildVersion, buildCommit, buildBranch, buildDate)
	if long {
		return "I am Groot " + line[len("groot "):]
	}
	return line
}

func versionPreRun(cmd *cobra.Command, _ []string) error {
	ver, _ := cmd.Flags().GetBool("version")
	if !ver {
		return nil
	}
	long, _ := cmd.Flags().GetBool("long")
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), FormatVersion(long)); err != nil {
		return err
	}
	if testing.Testing() {
		return ErrVersionPrinted
	}
	os.Exit(0)
	return nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, _ []string) error {
		long, _ := cmd.Flags().GetBool("long")
		_, err := fmt.Fprintln(cmd.OutOrStdout(), FormatVersion(long))
		return err
	},
}
