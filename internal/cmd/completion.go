package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// newCompletionCmd builds the `groot completion <shell>` subcommand.
//
// We deliberately expose only `bash`, `zsh`, `fish`, and `powershell` — Cobra's
// built-in `completion` subcommand silently accepts any shell name and prints
// help. For automation that pipes `groot completion zsh > _groot`, an unexpected
// shell argument writing nothing to stdout is unfriendly. The script is written
// to **stdout** so it can be redirected to a completion file; errors /
// unknown-shell messages go to **stderr** so they survive the same redirect.
func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion <bash|zsh|fish|powershell>",
		Short: "Generate a shell completion script for groot",
		Long: `Generate a shell completion script for the groot CLI.

Supported shells: bash, zsh, fish, powershell. The script is written to stdout
so you can install it with a redirect:

  groot completion bash | sudo install -m 0644 /dev/stdin /etc/bash_completion.d/groot
  groot completion zsh  > "${fpath[1]}/_groot"
  groot completion fish | source
  groot completion powershell | Out-File -Encoding utf8 $PROFILE.CurrentUserAllHosts`,
		Args: cobra.MatchAll(
			cobra.ExactArgs(1),
			func(_ *cobra.Command, args []string) error {
				if len(args) != 1 {
					return fmt.Errorf("expected exactly one shell name, got %d", len(args))
				}
				switch strings.ToLower(strings.TrimSpace(args[0])) {
				case "bash", "zsh", "fish", "powershell":
					return nil
				default:
					return fmt.Errorf("unsupported shell %q (expected one of: bash, zsh, fish, powershell)", args[0])
				}
			},
		),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := strings.ToLower(strings.TrimSpace(args[0]))
			out := cmd.OutOrStdout()

			// Prefer an *os.File when callers wire stdout to one (production
			// always does; tests may inject *bytes.Buffer — write through a
			// small adapter so we don't duplicate generator logic).
			w := out
			if _, ok := out.(*os.File); !ok {
				w = nonFileWriter{out}
			}

			switch shell {
			case "bash":
				return rootCmd.GenBashCompletion(w)
			case "zsh":
				return rootCmd.GenZshCompletion(w)
			case "fish":
				return rootCmd.GenFishCompletion(w, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletion(w)
			default:
				// Defensive — Args validator should reject everything above.
				fmt.Fprintf(cmd.ErrOrStderr(), "unsupported shell %q\n", shell)
				return fmt.Errorf("unsupported shell: %s", shell)
			}
		},
	}
}

// nonFileWriter adapts an io.Writer that is not *os.File (e.g. *bytes.Buffer
// from tests) so it can still feed Cobra's completion generators, which only
// require io.Writer. Drop-in; no allocation per call.
type nonFileWriter struct{ io.Writer }
