package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hrodrig/groot/internal/analyze"
	"github.com/hrodrig/groot/internal/arcread"
	"github.com/spf13/cobra"
)

// newAnalyzeCmd builds `groot analyze <archive>` (Phase 2 — offline heuristics).
// No kubeconfig or cluster access is required.
func newAnalyzeCmd() *cobra.Command {
	var outputForm string

	cmd := &cobra.Command{
		Use:   "analyze <archive.tar.gz>",
		Short: "Offline heuristic hints from a groot archive (executive Markdown)",
		Long: `Analyze opens a local ` + "`groot collect`" + ` archive (.tar.gz) and emits
evidence-backed heuristic hints (CrashLoopBackOff and related v1 scanners) as
executive Markdown by default. No cluster connection or kubeconfig is required.

Findings are hints and hypotheses based on offline evidence — not a definitive
root-cause diagnosis.

Exit codes follow #82:
  0  success (including zero hints / healthy empty summary)
  3  archive open or read failure`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arc, err := arcread.Open(args[0])
			if err != nil {
				return NewExitErrorf(ExitCollectAborted, "analyze archive: %v", err)
			}
			defer arc.Close()

			rep, err := analyze.Run(arc)
			if err != nil {
				return NewExitErrorf(ExitCollectAborted, "analyze archive: %v", err)
			}
			return renderAnalyze(cmd, rep, strings.ToLower(strings.TrimSpace(outputForm)))
		},
	}

	cmd.Flags().StringVar(&outputForm, "output", "text", "Output format: text (executive Markdown), json, or llm")
	return cmd
}

func renderAnalyze(cmd *cobra.Command, rep analyze.Report, format string) error {
	out := cmd.OutOrStdout()
	switch format {
	case "text", "":
		md, err := analyze.RenderExecutive(rep)
		if err != nil {
			return fmt.Errorf("render executive markdown: %w", err)
		}
		fmt.Fprint(out, md)
		return nil
	case "json":
		b, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal analyze report: %w", err)
		}
		fmt.Fprintln(out, string(b))
		return nil
	case "llm":
		md, err := analyze.RenderLLM(rep)
		if err != nil {
			return fmt.Errorf("render llm markdown: %w", err)
		}
		fmt.Fprint(out, md)
		return nil
	default:
		return fmt.Errorf("unsupported --output %q (want text, json, or llm)", format)
	}
}
