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
// When the binary was launched under the kubectl-groot basename (kubectl plugin
// dispatch), the binary name in the banner switches to kubectl-groot so logs
// can tell which entry point fired.
func FormatVersion(long bool) string {
	binary := "groot"
	if IsPluginInvocation() {
		binary = "kubectl-groot"
	}
	line := fmt.Sprintf("%s v%s (commit=%s branch=%s built=%s)",
		binary, buildVersion, buildCommit, buildBranch, buildDate)
	if long {
		// Strip the leading "<binary> " only; keep the rest identical so the
		// greeting still begins with "I am Groot" for both entries.
		greet := "I am Groot " + line[len(binary)+1:]
		return greet
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
