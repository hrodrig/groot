package notifier

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
)

// EventTest is sent by `groot notify test` (does not contact the Kubernetes API).
const EventTest = "notify.test"

// TestEvents are valid --event values for `groot notify test`.
var TestEvents = []string{EventTest, eventSuccess, eventFailure}

// ValidateTestEvent reports whether event is allowed for notify test.
func ValidateTestEvent(event string) error {
	switch strings.TrimSpace(event) {
	case EventTest, eventSuccess, eventFailure:
		return nil
	default:
		return fmt.Errorf(
			"notify test: unknown event %q (want: notify.test, success, failure)",
			event,
		)
	}
}

// AnyEnabled reports whether config has at least one notify destination.
func AnyEnabled(cfg config.Config) bool {
	return NewFanOut(cfg).Destinations() > 0
}

// DispatchTest posts event to all enabled channels without running collect.
func DispatchTest(ctx context.Context, cfg config.Config, event string) error {
	if err := ValidateTestEvent(event); err != nil {
		return err
	}
	fan := NewFanOut(cfg)
	if fan.Destinations() == 0 {
		return fmt.Errorf("notify test: no notify channel enabled in config")
	}

	sum := testSummary()
	switch strings.TrimSpace(event) {
	case eventFailure:
		return fan.NotifyFailure(ctx, sum, "notify test: simulated collect failure")
	case EventTest:
		return fan.notifyEvent(ctx, EventTest, sum, "")
	default:
		return fan.Notify(ctx, sum)
	}
}

func testSummary() collector.Summary {
	return collector.Summary{
		Total:       42,
		Success:     40,
		Failed:      2,
		Duration:    2 * time.Second,
		OutputDir:   "/tmp/groot-notify-test/out",
		ArchivePath: "/tmp/groot-notify-test/groot-notify-test.tar.gz",
		RunID:       "notify-test",
	}
}
