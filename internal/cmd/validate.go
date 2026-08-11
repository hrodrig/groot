package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
	"github.com/spf13/cobra"
)

// newValidateCmd builds `groot validate` (ROADMAP #31, #83 — 0.9.x).
//
// Pre-cluster collect: load config, run Preflight on the runtime, then exit.
// Failure categories:
//   - config / disk / RBAC error → exit 1 (config-tier failure)
//   - cluster / kubernetes error → exit 2 (API-tier failure)
//
// Flags:
//
//	--min-disk <bytes>        : override Collection.MinFreeBytes
//	--warn-disk <bytes>       : override Collection.WarnFreeBytes
//	--output <text|json>      : output format (default text)
func newValidateCmd() *cobra.Command {
	var (
		minDisk    int64
		warnDisk   int64
		outputForm string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Preflight checks (config + cluster + RBAC + disk) without collecting",
		Long: `Validate that a ` + "`groot collect`" + ` run would succeed without writing any output.

Checks performed:
  - config YAML loads and parses
  - Kubernetes API handshake (cluster-info)
  - RBAC matrix for the verbs the collector uses (auth can-i)
  - free disk space on output_dir (hard fail + warn thresholds)

Exit codes follow #82:
  0  all checks passed (warnings allowed)
  1  config / disk / RBAC failure
  2  Kubernetes client / API failure`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return NewExitError(ExitConfigError, fmt.Errorf("load config: %w", err))
			}
			applyKubeconfigFlag(&cfg)
			if minDisk > 0 {
				cfg.Collection.MinFreeBytes = minDisk
			}
			if warnDisk > 0 {
				cfg.Collection.WarnFreeBytes = warnDisk
			}
			if strings.TrimSpace(cfg.OutputDir) == "" {
				cfg.OutputDir = "./groot-out"
			}
			// Load already expands ~/ and ${VAR}; keep abs resolution for preflight.
			if strings.HasPrefix(cfg.OutputDir, "~/") {
				cfg.OutputDir = config.ExpandPath(cfg.OutputDir)
			}
			if !filepath.IsAbs(cfg.OutputDir) {
				if cwd, cerr := os.Getwd(); cerr == nil {
					cfg.OutputDir = filepath.Join(cwd, cfg.OutputDir)
				}
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			svc := collector.New(cfg)
			result := svc.Preflight(ctx)

			renderValidate(cmd, result, strings.ToLower(strings.TrimSpace(outputForm)))

			if !result.OK {
				// Inspect the findings to pick the right exit code (config vs k8s).
				for _, f := range result.Findings {
					if f.Severity != collector.PreflightError {
						continue
					}
					if strings.HasPrefix(f.Check, "kubernetes.") {
						return NewExitError(ExitKubernetesError, fmt.Errorf("preflight: %s", f.Message))
					}
				}
				return NewExitError(ExitConfigError, fmt.Errorf("preflight failed"))
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&minDisk, "min-disk", 0, "Minimum free bytes required on output_dir (override; 0 = default 256MiB)")
	cmd.Flags().Int64Var(&warnDisk, "warn-disk", 0, "Warn threshold for free bytes on output_dir (override; 0 = default 1GiB)")
	cmd.Flags().StringVar(&outputForm, "output", "text", "Output format: text or json")

	return cmd
}

// newInspectCmd builds `groot inspect <archive>` (ROADMAP #31 minimum — 0.9.x).
// Prints the manifest, the sorted file tree, and total counts. No cluster access.
func newInspectCmd() *cobra.Command {
	var (
		outputForm      string
		maxDecompressed int64
	)

	cmd := &cobra.Command{
		Use:   "inspect <archive.tar.gz>",
		Short: "Inspect an existing groot archive (manifest + file tree)",
		Long: `Inspect opens a ` + "`groot collect`" + ` archive (.tar.gz), reads the manifest if
present, and lists the file tree with sizes. No cluster connection is required.

Archive open enforces arcread safety caps (default max decompressed total 16GiB).
Override with --max-decompressed when indexing a larger capture.

Exit codes follow #82: 0 success, 3 archive read failure, 1 parse failure.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			caps := archiveCapsFromMaxDecompressed(maxDecompressed)
			info, err := collector.InspectArchiveWithCaps(args[0], caps)
			if err != nil {
				return NewExitErrorf(ExitCollectAborted, "inspect archive: %v", err)
			}
			renderInspect(cmd, info, strings.ToLower(strings.TrimSpace(outputForm)))

			// If we found files but no manifest.json, signal it via parse_err
			// (don't change exit code — this is informational and won't break
			// downstream scripting in 0.9.x; #87 golden fixtures arrive later).
			return nil
		},
	}

	cmd.Flags().StringVar(&outputForm, "output", "text", "Output format: text or json")
	cmd.Flags().Int64Var(&maxDecompressed, "max-decompressed", 0, "Max total decompressed archive bytes during index (0 = default 16GiB)")
	return cmd
}

func renderValidate(cmd *cobra.Command, r collector.PreflightResult, format string) {
	out := cmd.OutOrStdout()
	if format == "json" {
		// json.MarshalIndent lives in the stdlib; cheap and safe.
		b, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "marshal preflight result: %v\n", err)
			return
		}
		fmt.Fprintln(out, string(b))
		return
	}

	fmt.Fprintf(out, "Preflight: %s\n", verdict(r.OK))
	if r.Cluster != "" {
		fmt.Fprintf(out, "  cluster: %s\n", r.Cluster)
	}
	if r.OutputDir != "" {
		fmt.Fprintf(out, "  output_dir: %s\n", r.OutputDir)
	}
	fmt.Fprintf(out, "  findings: %d\n", len(r.Findings))
	for _, f := range r.Findings {
		fmt.Fprintf(out, "  - [%s] %s: %s\n", severityLabel(f.Severity), f.Check, f.Message)
	}
}

func renderInspect(cmd *cobra.Command, info collector.InspectInfo, format string) {
	out := cmd.OutOrStdout()
	if format == "json" {
		b, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "marshal inspect info: %v\n", err)
			return
		}
		fmt.Fprintln(out, string(b))
		return
	}

	fmt.Fprintf(out, "Archive: %s (%s bytes)\n", info.ArchivePath, humanBytes(info.ArchiveSize))
	fmt.Fprintf(out, "Files: %d\n", info.FileCount)
	for _, f := range info.Files {
		fmt.Fprintf(out, "  %s\n", f)
	}
	if info.ManifestJSON != "" {
		fmt.Fprintf(out, "\nManifest:\n%s\n", indent(info.ManifestJSON, "  "))
	}
	if info.ParseErr != "" {
		fmt.Fprintf(out, "\nParse warnings: %s\n", info.ParseErr)
	}
}

func verdict(ok bool) string {
	if ok {
		return "OK"
	}
	return "FAIL"
}

func severityLabel(s collector.PreflightSeverity) string {
	switch s {
	case collector.PreflightError:
		return "ERROR"
	case collector.PreflightWarn:
		return "WARN "
	case collector.PreflightOK:
		return "OK   "
	}
	return "????"
}

func humanBytes(n int64) string {
	return collectorFormatBytes(n)
}

// collectorFormatBytes delegates to a local helper to avoid coupling two
// packages around the same format constants. The collector package owns the
// canonical byte formatter.
func collectorFormatBytes(n int64) string {
	const (
		KiB = 1024
		MiB = 1024 * 1024
		GiB = 1024 * 1024 * 1024
	)
	switch {
	case n >= GiB:
		return fmt.Sprintf("%.1fGiB", float64(n)/float64(GiB))
	case n >= MiB:
		return fmt.Sprintf("%.1fMiB", float64(n)/float64(MiB))
	case n >= KiB:
		return fmt.Sprintf("%.1fKiB", float64(n)/float64(KiB))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

func indent(s, pad string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		b.WriteString(pad)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
