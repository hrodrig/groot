package notifier

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hrodrig/groot/internal/collector"
)

const (
	slackColorGood    = "good"    // green — success / notify.test
	slackColorWarning = "warning" // yellow — partial collect failure
	slackColorDanger  = "danger"  // red — abort / hard failure
)

type slackWebhookPayload struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Color    string   `json:"color"`
	Fallback string   `json:"fallback"`
	Text     string   `json:"text"`
	Footer   string   `json:"footer,omitempty"`
	Ts       int64    `json:"ts,omitempty"`
	Markdown []string `json:"mrkdwn_in,omitempty"`
}

func buildSlackPayload(ctx messageContext) slackWebhookPayload {
	line := ctx.SummaryLine
	att := slackAttachment{
		Color:    slackColorFor(ctx),
		Fallback: line,
		Text:     slackAttachmentBody(ctx),
		Footer:   slackFooter(ctx.Summary),
		Ts:       slackTs(ctx.Summary),
		Markdown: []string{"text"},
	}
	return slackWebhookPayload{
		Text:        line,
		Attachments: []slackAttachment{att},
	}
}

// slackColorFor maps notify event → Slack attachment vertical bar color.
// success / notify.test → good (green); partial failure (no reason) → warning
// (yellow); abort / hard failure (reason set) → danger (red).
func slackColorFor(ctx messageContext) string {
	if ctx.Event != eventFailure {
		return slackColorGood
	}
	if strings.TrimSpace(ctx.Reason) == "" {
		return slackColorWarning
	}
	return slackColorDanger
}

func slackAttachmentBody(ctx messageContext) string {
	sum := ctx.Summary
	var b strings.Builder
	b.WriteString(slackTitle(ctx.Event))
	b.WriteByte('\n')
	writeSlackBullet(&b, "Cluster", sum.Cluster)
	writeSlackBullet(&b, "Client", sum.ClientID)
	writeSlackBullet(&b, "Time", slackTime(sum))
	writeSlackBullet(&b, "Context", sum.Context)
	writeSlackBullet(&b, "User", sum.User)
	writeSlackBullet(&b, "Mode", sum.Mode)
	writeSlackBullet(&b, "Duration", sum.Duration.Round(time.Second).String())
	writeSlackBullet(&b, "Jobs", slackJobs(sum))
	if base := filepath.Base(strings.TrimSpace(sum.ArchivePath)); base != "" && base != "." {
		writeSlackBullet(&b, "Archive", base)
	}
	writeSlackBullet(&b, "Run ID", sum.RunID)
	if ctx.Event == eventFailure {
		if reason := strings.TrimSpace(ctx.Reason); reason != "" {
			writeSlackBullet(&b, "Reason", reason)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func slackTitle(event string) string {
	switch event {
	case EventTest:
		return "✅ *GROOT notify test*"
	case eventFailure:
		return "❌ *GROOT failed*"
	default:
		return "✅ *GROOT completed*"
	}
}

func writeSlackBullet(b *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	fmt.Fprintf(b, "• *%s:* `%s`\n", label, value)
}

func slackJobs(sum collector.Summary) string {
	if sum.Total == 0 && sum.Success == 0 && sum.Failed == 0 {
		return ""
	}
	return fmt.Sprintf("total=%d success=%d failed=%d", sum.Total, sum.Success, sum.Failed)
}

func slackTime(sum collector.Summary) string {
	if sum.FinishedAt.IsZero() {
		return ""
	}
	return sum.FinishedAt.UTC().Format(time.RFC3339)
}

func slackFooter(sum collector.Summary) string {
	ver := strings.TrimSpace(sum.Version)
	if ver == "" {
		ver = "dev"
	}
	if !strings.HasPrefix(ver, "v") && ver != "dev" {
		ver = "v" + ver
	}
	footer := "groot " + ver
	if !sum.FinishedAt.IsZero() {
		footer += " | " + sum.FinishedAt.UTC().Format("2 Jan")
	}
	return footer
}

func slackTs(sum collector.Summary) int64 {
	if sum.FinishedAt.IsZero() {
		return time.Now().UTC().Unix()
	}
	return sum.FinishedAt.UTC().Unix()
}
