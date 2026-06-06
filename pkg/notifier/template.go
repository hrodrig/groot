package notifier

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/groot/pkg/collector"
)

type messageContext struct {
	Event       string
	SummaryLine string
	Reason      string
	Summary     collector.Summary
}

func buildSummaryLine(event string, sum collector.Summary, reason string) string {
	duration := sum.Duration.Round(time.Second)
	switch event {
	case eventFailure:
		if strings.TrimSpace(reason) != "" {
			return fmt.Sprintf(
				"GROOT FAILED. reason=%s total=%d success=%d failed=%d duration=%s output=%s archive=%s",
				reason, sum.Total, sum.Success, sum.Failed, duration, sum.OutputDir, sum.ArchivePath,
			)
		}
		return fmt.Sprintf(
			"GROOT finished with failures. total=%d success=%d failed=%d duration=%s output=%s archive=%s",
			sum.Total, sum.Success, sum.Failed, duration, sum.OutputDir, sum.ArchivePath,
		)
	default:
		return fmt.Sprintf(
			"GROOT finished. total=%d success=%d failed=%d duration=%s output=%s archive=%s",
			sum.Total, sum.Success, sum.Failed, duration, sum.OutputDir, sum.ArchivePath,
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
	}
	out := tmpl
	for k, v := range repl {
		out = strings.ReplaceAll(out, k, v)
	}
	return out
}
