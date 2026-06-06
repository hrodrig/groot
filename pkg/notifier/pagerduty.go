package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hrodrig/groot/pkg/collector"
)

// pagerDutyEventsURL is the Events API v2 enqueue endpoint (overridable in tests).
var pagerDutyEventsURL = "https://events.pagerduty.com/v2/enqueue"

type pagerDutyRoute struct {
	routingKey string
	severity   string
	source     string
}

func sendPagerDutyV2(ctx context.Context, route pagerDutyRoute, summaryLine string, sum collector.Summary) error {
	payload := map[string]any{
		"summary":   summaryLine,
		"source":    route.source,
		"severity":  route.severity,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"custom_details": map[string]any{
			"total":        sum.Total,
			"success":      sum.Success,
			"failed":       sum.Failed,
			"duration":     sum.Duration.Round(time.Second).String(),
			"output_dir":   sum.OutputDir,
			"archive_path": sum.ArchivePath,
		},
	}
	envelope := map[string]any{
		"routing_key":  route.routingKey,
		"event_action": "trigger",
		"payload":      payload,
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal pagerduty event: %w", err)
	}

	if err := postJSONWithRetry(ctx, pagerDutyEventsURL, body, nil); err != nil {
		return fmt.Errorf("send pagerduty event: %w", err)
	}
	return nil
}
