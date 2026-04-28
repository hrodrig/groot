package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"groot/pkg/collector"
	"groot/pkg/config"
	"groot/pkg/logx"
	"groot/pkg/notifier"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var cfgFile string
var printSampleConfig bool
var verbose bool
var quiet bool
var noColor bool
var message string
var kubeconfigOverride string
var buildVersion = "dev"
var buildCommit = "unknown"
var buildBranch = "unknown"
var buildDate = "unknown"
var testConnection bool

var rootCmd = &cobra.Command{
	Use:   "groot",
	Short: "Collect Kubernetes logs and diagnostics",
	Long:  "groot collects as many Kubernetes logs and diagnostics as possible for worker nodes, control plane components, and workloads.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if printSampleConfig {
			cmd.Print(config.SampleYAML())
			return nil
		}
		if testConnection {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if kubeconfigOverride != "" {
				cfg.Kubeconfig = kubeconfigOverride
			}
			meta, err := runConnectionTest(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			if !quiet {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"Kubernetes connection: OK (context=%s cluster=%s server=%s)\n",
					meta.Context,
					meta.Cluster,
					meta.Server,
				)
			}
			return nil
		}
		return cmd.Help()
	},
}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect cluster logs",
	RunE: func(cmd *cobra.Command, _ []string) error {
		logger := logx.New(verbose, quiet, noColor)

		if printSampleConfig {
			cmd.Print(config.SampleYAML())
			return nil
		}

		if _, err := exec.LookPath("kubectl"); err != nil {
			logger.Error("kubectl not found in PATH; install kubectl and retry")
			return fmt.Errorf("kubectl not found in PATH: %w", err)
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if kubeconfigOverride != "" {
			cfg.Kubeconfig = kubeconfigOverride
		}
		if testConnection {
			meta, err := runConnectionTest(cmd.Context(), cfg)
			if err != nil {
				logger.Error("kubernetes connection test failed: %v", err)
				return err
			}
			if !quiet {
				logger.Info(
					"kubernetes connection test passed (context=%s cluster=%s server=%s)",
					meta.Context,
					meta.Cluster,
					meta.Server,
				)
			}
			return nil
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Collection.Timeout)
		defer cancel()

		collectorSvc := collector.New(cfg)
		collectorSvc.SetMessage(message)
		collectorSvc.SetHooks(
			func(name string, args []string) {
				logger.Cmd("%s -> kubectl %s", name, strings.Join(args, " "))
			},
			func(name string) {
				logger.OK("%s completed", name)
			},
			func(name string, err error) {
				logger.Error("%s failed: %v", name, err)
			},
		)
		summary, err := collectorSvc.Run(ctx)
		if err != nil {
			logger.Error("collection failed: %v", err)
			return fmt.Errorf("collect logs: %w", err)
		}

		notifierSvc := notifier.NewFanOut(cfg)
		if notifyErr := notifierSvc.Notify(ctx, summary); notifyErr != nil {
			logger.Error("notification failed: %v", notifyErr)
			return fmt.Errorf("send notifications: %w", notifyErr)
		}

		if !quiet {
			logger.Info("collection completed in %s", summary.Duration.Round(time.Second))
			logger.Info("output dir: %s", summary.OutputDir)
			logger.Info("archive: %s", summary.ArchivePath)
			logger.Info("commands: total=%d success=%d failed=%d", summary.Total, summary.Success, summary.Failed)
		}

		if summary.Failed > 0 {
			for _, failure := range summary.Failures {
				logger.Warn("command failure: %s", failure)
			}
		}
		return nil
	},
}

// Execute runs the CLI.
func Execute() error {
	return rootCmd.Execute()
}

// ResetPersistentCLI restores root persistent flags to defaults (for tests calling Execute from main package).
func ResetPersistentCLI() {
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// SetBuildInfo injects build metadata used by --version.
func SetBuildInfo(version, commit, branch, date string) {
	buildVersion = version
	buildCommit = commit
	buildBranch = branch
	buildDate = date
	rootCmd.Version = fmt.Sprintf("%s (commit=%s branch=%s built=%s)", buildVersion, buildCommit, buildBranch, buildDate)
}

func init() {
	rootCmd.Version = fmt.Sprintf("%s (commit=%s branch=%s built=%s)", buildVersion, buildCommit, buildBranch, buildDate)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Path to YAML config file")
	rootCmd.PersistentFlags().BoolVar(&printSampleConfig, "print-sample-config", false, "Print sample groot.yml and exit")
	rootCmd.PersistentFlags().BoolVar(&testConnection, "test-connection", false, "Validate Kubernetes connectivity and exit")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable detailed command execution output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress normal output, show errors only")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colorized console output")
	rootCmd.PersistentFlags().StringVar(&message, "message", "", "Custom suffix appended to capture output names")
	rootCmd.PersistentFlags().StringVar(&kubeconfigOverride, "kubeconfig", "", "Override kubeconfig path for kubectl commands")
	rootCmd.AddCommand(collectCmd)
}

type connMeta struct {
	Context string
	Cluster string
	Server  string
}

func runConnectionTest(ctx context.Context, cfg config.Config) (connMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	collectorSvc := collector.New(cfg)
	meta, err := collectorSvc.ReadKubeMetadata(ctx)
	if err != nil {
		return connMeta{}, fmt.Errorf("read kube metadata: %w", err)
	}

	argsFor := func(args []string) []string {
		if cfg.Kubeconfig == "" {
			return args
		}
		return append([]string{"--kubeconfig", cfg.Kubeconfig}, args...)
	}

	if err := exec.CommandContext(ctx, "kubectl", argsFor([]string{"config", "current-context"})...).Run(); err != nil {
		return connMeta{}, fmt.Errorf("resolve current context: %w", err)
	}
	if err := exec.CommandContext(ctx, "kubectl", argsFor([]string{"get", "ns", "--request-timeout=10s", "-o", "name"})...).Run(); err != nil {
		return connMeta{}, fmt.Errorf("list namespaces: %w", err)
	}
	return connMeta{
		Context: valueOrUnknown(meta.Context),
		Cluster: valueOrUnknown(meta.Cluster),
		Server:  valueOrUnknown(meta.Server),
	}, nil
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
