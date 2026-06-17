package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hrodrig/groot/pkg/collector"
	"github.com/hrodrig/groot/pkg/config"
	"github.com/hrodrig/groot/pkg/kubeloader"
	"github.com/hrodrig/groot/pkg/logx"
	"github.com/hrodrig/groot/pkg/notifier"
	"github.com/hrodrig/groot/pkg/uploader"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var cfgFile string
var printSampleConfig bool
var verbose bool
var quiet bool
var noNotify bool
var noUpload bool
var noColor bool
var message string
var kubeconfigOverride string
var buildVersion = "dev"
var buildCommit = "unknown"
var buildBranch = "unknown"
var buildDate = "unknown"
var testConnection bool
var collectLogsSince string
var listJobs bool

var rootCmd = &cobra.Command{
	Use:   "groot",
	Short: "Collect Kubernetes logs and cluster context",
	Long:  "Groot collects read-only Kubernetes logs, events, and selected API snapshots into one archive (client-go; no kubectl binary required).",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if printSampleConfig {
			// Write to stdout so shell redirects (`> file`) work. Cobra's cmd.Print uses stderr.
			_, err := fmt.Fprint(cmd.OutOrStdout(), config.SampleYAML())
			return err
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
			// Write to stdout so shell redirects (`> file`) work. Cobra's cmd.Print uses stderr.
			_, err := fmt.Fprint(cmd.OutOrStdout(), config.SampleYAML())
			return err
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cmd.Flags().Lookup("since").Changed {
			norm, normErr := config.NormalizePodLogsSince(collectLogsSince)
			if normErr != nil {
				return fmt.Errorf("invalid --since: %w", normErr)
			}
			cfg.Collection.PodLogsSince = norm
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

		collectorSvc := collector.New(cfg)
		collectorSvc.SetBuildInfo(buildVersion, buildCommit, buildBranch, buildDate)
		collectorSvc.SetMessage(message)

		if listJobs {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Collection.Timeout)
			defer cancel()
			plans, err := collectorSvc.ListJobs(ctx)
			if err != nil {
				return fmt.Errorf("list jobs: %w", err)
			}
			for _, p := range plans {
				opt := ""
				if p.Optional {
					opt = " optional"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s%s args=%v\n", p.Name, p.FileName, opt, p.Args)
			}
			if !quiet {
				logger.Info("planned jobs: %d", len(plans))
			}
			return nil
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Collection.Timeout)
		defer cancel()
		collectorSvc.SetHooks(
			func(name string, args []string) {
				logger.Cmd("%s -> %s", name, strings.Join(args, " "))
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
			if !skipNotifications() && notifier.ShouldNotifyAbort(cfg) {
				notifierSvc := notifier.NewFanOut(cfg)
				if notifyErr := notifierSvc.NotifyFailure(ctx, summary, err.Error()); notifyErr != nil {
					logger.Error("failure notification failed: %v", notifyErr)
					return fmt.Errorf("send failure notifications: %w", notifyErr)
				}
			}
			return fmt.Errorf("collect logs: %w", err)
		}

		if !skipNotifications() {
			notifierSvc := notifier.NewFanOut(cfg)
			if notifyErr := notifierSvc.Notify(ctx, summary); notifyErr != nil {
				logger.Error("notification failed: %v", notifyErr)
				return fmt.Errorf("send notifications: %w", notifyErr)
			}
			if notifier.ShouldNotifyPartialFailure(cfg, summary) {
				if notifyErr := notifierSvc.NotifyFailure(ctx, summary, ""); notifyErr != nil {
					logger.Error("failure notification failed: %v", notifyErr)
					return fmt.Errorf("send failure notifications: %w", notifyErr)
				}
			}
		}

		if summary.ArchivePath != "" && uploader.ShouldUpload(cfg) && !skipUploads() {
			uploadSvc := uploader.NewFanOut(cfg)
			for _, outcome := range uploadSvc.Upload(ctx, summary.ArchivePath, summary) {
				if outcome.Err != nil {
					logger.Error("%s upload failed: %v", outcome.Provider, outcome.Err)
					continue
				}
				logger.OK("%s", uploader.FormatResult(outcome.Result))
			}
		}

		if !quiet {
			logger.Info("collection completed in %s", summary.Duration.Round(time.Second))
			logger.Info("output dir: %s", summary.OutputDir)
			logger.Info("archive: %s", summary.ArchivePath)
			logger.Info("jobs: total=%d success=%d failed=%d", summary.Total, summary.Success, summary.Failed)
		}

		if summary.Failed > 0 {
			for _, failure := range summary.Failures {
				logger.Warn("job failure: %s", failure)
			}
		}
		return nil
	},
}

// Execute runs the CLI.
func Execute() error {
	err := rootCmd.Execute()
	if errors.Is(err, ErrVersionPrinted) {
		return nil
	}
	return err
}

// ResetPersistentCLI restores root and collect-command flags to defaults (for tests calling Execute from main package).
func ResetPersistentCLI() {
	resetFlags := func(cmd *cobra.Command) {
		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
	resetFlags(rootCmd)
	resetFlags(collectCmd)
}

// SetBuildInfo injects build metadata used by --version.
func SetBuildInfo(version, commit, branch, date string) {
	buildVersion = version
	buildCommit = commit
	buildBranch = branch
	buildDate = date
}

func init() {
	rootCmd.PersistentPreRunE = versionPreRun
	initVersionFlags(rootCmd)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Path to YAML config file")
	rootCmd.PersistentFlags().BoolVar(&printSampleConfig, "print-sample-config", false, "Print sample groot.yml and exit")
	rootCmd.PersistentFlags().BoolVar(&testConnection, "test-connection", false, "Validate Kubernetes connectivity and exit")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable detailed command execution output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress normal console output (INFO/WARN/CMD/OK); does not affect webhooks or other notify integrations")
	rootCmd.PersistentFlags().BoolVar(&noNotify, "no-notify", false, "Skip all notify integrations after collect (Slack, Discord, Teams, PagerDuty, Telegram, generic). Also honored when env GROOT_NO_NOTIFY is 1/true/yes")
	rootCmd.PersistentFlags().BoolVar(&noUpload, "no-upload", false, "Skip post-collect archive upload (S3/GCS). Also honored when env GROOT_NO_UPLOAD is 1/true/yes")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colorized console output")
	rootCmd.PersistentFlags().StringVar(&message, "message", "", "Custom suffix appended to capture output names")
	rootCmd.PersistentFlags().StringVar(&kubeconfigOverride, "kubeconfig", "", "Override kubeconfig path for the Kubernetes API client")
	collectCmd.Flags().StringVar(&collectLogsSince, "since", "", "Pod logs only: --since duration (e.g. 24h, 45m; bare number means hours). Overrides collection.pod_logs_since in config when set")
	collectCmd.Flags().BoolVar(&listJobs, "list-jobs", false, "Print planned collection jobs and exit without writing output")
	rootCmd.AddCommand(collectCmd, versionCmd)
}

type connMeta struct {
	Context string
	Cluster string
	Server  string
}

func runConnectionTest(ctx context.Context, cfg config.Config) (connMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rc, err := kubeloader.RESTConfig(cfg.Kubeconfig)
	if err != nil {
		return connMeta{}, fmt.Errorf("kubeconfig: %w", err)
	}
	cs, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return connMeta{}, fmt.Errorf("kubernetes client: %w", err)
	}
	if _, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1}); err != nil {
		return connMeta{}, fmt.Errorf("list namespaces: %w", err)
	}

	collectorSvc := collector.New(cfg)
	meta, err := collectorSvc.ReadKubeMetadata(ctx)
	if err != nil {
		return connMeta{}, fmt.Errorf("read kube metadata: %w", err)
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

func skipNotifications() bool {
	if noNotify {
		return true
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GROOT_NO_NOTIFY")))
	return v == "1" || v == "true" || v == "yes"
}

func skipUploads() bool {
	if noUpload {
		return true
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GROOT_NO_UPLOAD")))
	return v == "1" || v == "true" || v == "yes"
}
