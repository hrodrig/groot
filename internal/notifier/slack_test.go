package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
)

func TestBuildSlackPayload_success(t *testing.T) {
	sum := collector.Summary{
		Total:       10,
		Success:     10,
		Failed:      0,
		Duration:    14 * time.Second,
		ArchivePath: "/var/log/groot/capture-cluster.tar.gz",
		RunID:       "run-1",
		ClientID:    "bastion-01",
		Cluster:     "prod-cluster",
		Context:     "prod-ctx",
		User:        "ops",
		Mode:        "collect",
		Version:     "1.2.2",
		FinishedAt:  time.Date(2026, 11, 5, 13, 14, 4, 0, time.UTC),
	}
	line := buildSummaryLine(eventSuccess, sum, "")
	p := buildSlackPayload(messageContext{Event: eventSuccess, Summary: sum, SummaryLine: line})
	if p.Text != "" {
		t.Fatalf("top-level text must be empty (avoid Slack duplicate), got %q", p.Text)
	}
	if len(p.Attachments) != 1 {
		t.Fatalf("attachments=%d", len(p.Attachments))
	}
	att := p.Attachments[0]
	if att.Color != slackColorGood {
		t.Fatalf("color=%q", att.Color)
	}
	if att.Fallback != line {
		t.Fatalf("fallback=%q", att.Fallback)
	}
	for _, want := range []string{
		"✅ *GROOT completed*",
		"*Cluster:* `prod-cluster`",
		"*Client:* `bastion-01`",
		"*Time:* `2026-11-05T13:14:04Z`",
		"*Context:* `prod-ctx`",
		"*User:* `ops`",
		"*Mode:* `collect`",
		"*Duration:* `14s`",
		"*Jobs:* `total=10 success=10 failed=0`",
		"*Archive:* `capture-cluster.tar.gz`",
		"*Run ID:* `run-1`",
	} {
		if !strings.Contains(att.Text, want) {
			t.Fatalf("attachment text missing %q\n%s", want, att.Text)
		}
	}
	if att.Footer != "groot v1.2.2 | 5 Nov" {
		t.Fatalf("footer=%q", att.Footer)
	}
	if att.Ts != sum.FinishedAt.Unix() {
		t.Fatalf("ts=%d", att.Ts)
	}
}

func TestBuildSlackPayload_failureOmitsEmpty(t *testing.T) {
	sum := collector.Summary{
		Total:    3,
		Success:  1,
		Failed:   2,
		Duration: time.Second,
		Version:  "dev",
	}
	line := buildSummaryLine(eventFailure, sum, "boom")
	p := buildSlackPayload(messageContext{Event: eventFailure, Summary: sum, Reason: "boom", SummaryLine: line})
	att := p.Attachments[0]
	if att.Color != slackColorDanger {
		t.Fatalf("color=%q want danger", att.Color)
	}
	if !strings.Contains(att.Text, "❌ *GROOT failed*") {
		t.Fatalf("title missing: %s", att.Text)
	}
	if !strings.Contains(att.Text, "*Reason:* `boom`") {
		t.Fatalf("reason missing: %s", att.Text)
	}
	if strings.Contains(att.Text, "*Client:*") || strings.Contains(att.Text, "*Cluster:*") {
		t.Fatalf("empty fields should omit: %s", att.Text)
	}
	if att.Footer != "groot dev" {
		t.Fatalf("footer=%q", att.Footer)
	}
}

func TestSlackJobs_empty(t *testing.T) {
	if got := slackJobs(collector.Summary{}); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestSendSlack_marshalError(t *testing.T) {
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(any) ([]byte, error) {
		return nil, errors.New("boom")
	}
	w := &webhookSender{url: "http://127.0.0.1:0/unused", kind: "slack", http: testHTTPNotify()}
	err := w.SendSlack(context.Background(), messageContext{Event: eventSuccess, SummaryLine: "x"})
	if err == nil || !strings.Contains(err.Error(), "marshal slack") {
		t.Fatalf("err=%v", err)
	}
}

func TestSlackColorFor(t *testing.T) {
	if c := slackColorFor(messageContext{Event: eventSuccess}); c != slackColorGood {
		t.Fatalf("success=%q", c)
	}
	if c := slackColorFor(messageContext{Event: EventTest}); c != slackColorGood {
		t.Fatalf("test=%q", c)
	}
	if c := slackColorFor(messageContext{Event: eventFailure}); c != slackColorWarning {
		t.Fatalf("partial=%q", c)
	}
	if c := slackColorFor(messageContext{Event: eventFailure, Reason: "archive failed"}); c != slackColorDanger {
		t.Fatalf("abort=%q", c)
	}
}

func TestFanOut_Notify_slackRichAttachment(t *testing.T) {
	var raw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Config{}
	cfg.Notify.Slack.Enabled = true
	cfg.Notify.Slack.WebhookURL = srv.URL
	f := NewFanOut(cfg)
	sum := testSummary()
	sum.ClientID = "bastion-01"
	if err := f.Notify(context.Background(), sum); err != nil {
		t.Fatal(err)
	}
	var p slackWebhookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("json: %v body=%s", err, raw)
	}
	if p.Text != "" || len(p.Attachments) != 1 {
		t.Fatalf("payload=%#v", p)
	}
	if p.Attachments[0].Fallback == "" {
		t.Fatal("expected fallback one-liner")
	}
	if !strings.Contains(p.Attachments[0].Text, "*Client:* `bastion-01`") {
		t.Fatalf("attachment=%s", p.Attachments[0].Text)
	}
}
