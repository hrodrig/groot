package cmd

import (
	"fmt"

	"github.com/hrodrig/groot/internal/config"
	"github.com/hrodrig/groot/internal/notifier"
	"github.com/spf13/cobra"
)

func newNotifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify",
		Short: "Outbound notification helpers",
	}
	cmd.AddCommand(newNotifyTestCmd())
	return cmd
}

func newNotifyTestCmd() *cobra.Command {
	var event string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Send a test notification without running collect",
		Long: `Loads notify.* from config and sends a synthetic summary to every enabled channel.
Does not contact the Kubernetes API or write an archive.

Default event is notify.test. Use --event to preview success or failure payload formatting.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return NewExitError(ExitConfigError, fmt.Errorf("load config: %w", err))
			}
			if err := notifier.ValidateTestEvent(event); err != nil {
				return NewExitError(ExitConfigError, err)
			}
			if !notifier.AnyEnabled(cfg) {
				return NewExitError(ExitConfigError, fmt.Errorf("notify test: no notify channel enabled in config"))
			}
			if err := notifier.DispatchTest(cmd.Context(), cfg, event); err != nil {
				return NewExitError(ExitNotifyFailed, fmt.Errorf("notify test: %w", err))
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "notify test: sent event %q to enabled channel(s)\n", event)
			return err
		},
	}
	cmd.Flags().StringVar(&event, "event", notifier.EventTest, "event to send: notify.test, success, failure")
	return cmd
}
