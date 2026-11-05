package notifier

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/groot/internal/collector"
)

type messageContext struct {
	Event       string
	SummaryLine string
	Reason      string
	Summary     collector.Summary
}

func clientAttr(sum collector.Summary) string {
	if id := strings.TrimSpace(sum.ClientID); id != "" {
		return fmt.Sprintf(" client=%s", id)
	}
	return ""
}

func buildSummaryLine(event string, sum collector.Summary, reason string) string {
	duration := sum.Duration.Round(time.Second)
	client := clientAttr(sum)
	switch event {
	case EventTest:
		return fmt.Sprintf(
			"GROOT notify test.%s total=%d success=%d failed=%d duration=%s output=%s archive=%s run_id=%s",
			client, sum.Total, sum.Success, sum.Failed, duration, sum.OutputDir, sum.ArchivePath, sum.RunID,
		)
	case eventFailure:
		if strings.TrimSpace(reason) != "" {
			return fmt.Sprintf(
				"GROOT FAILED.%s reason=%s total=%d success=%d failed=%d duration=%s output=%s archive=%s",
				client, reason, sum.Total, sum.Success, sum.Failed, duration, sum.OutputDir, sum.ArchivePath,
			)
		}
		return fmt.Sprintf(
			"GROOT finished with failures.%s total=%d success=%d failed=%d duration=%s output=%s archive=%s",
			client, sum.Total, sum.Success, sum.Failed, duration, sum.OutputDir, sum.ArchivePath,
		)
	default:
		return fmt.Sprintf(
			"GROOT finished.%s total=%d success=%d failed=%d duration=%s output=%s archive=%s",
			client, sum.Total, sum.Success, sum.Failed, duration, sum.OutputDir, sum.ArchivePath,
		)
	}
}

func applyTemplate(tmpl string, ctx messageContext) string {
	repl := map[string]string{
		"{{event}}":        ctx.Event,
		"{{summary}}":      ctx.SummaryLine,
		"{{text}}":         ctx.SummaryLine,
		"{{reason}}":       ctx.Reason,
		"{{total}}":        strconv.Itoa(ctx.Summary.Total),
		"{{success}}":      strconv.Itoa(ctx.Summary.Success),
		"{{failed}}":       strconv.Itoa(ctx.Summary.Failed),
		"{{duration}}":     ctx.Summary.Duration.Round(time.Second).String(),
		"{{output_dir}}":   ctx.Summary.OutputDir,
		"{{archive_path}}": ctx.Summary.ArchivePath,
		"{{run_id}}":       ctx.Summary.RunID,
		"{{client_id}}":    strings.TrimSpace(ctx.Summary.ClientID),
	}
	out := tmpl
	for k, v := range repl {
		out = strings.ReplaceAll(out, k, v)
	}
	return out
}
