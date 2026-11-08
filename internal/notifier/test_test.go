package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/config"
)

func TestValidateTestEvent(t *testing.T) {
	for _, ev := range TestEvents {
		if err := ValidateTestEvent(ev); err != nil {
			t.Fatalf("event %q: %v", ev, err)
		}
	}
	if err := ValidateTestEvent("bogus"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAnyEnabled(t *testing.T) {
	if AnyEnabled(config.Config{}) {
		t.Fatal("empty config should have no destinations")
	}
	cfg := config.Config{
		Notify: config.NotifyCfg{
			Slack: config.WebhookCfg{Enabled: true, WebhookURL: "http://example.com/hook"},
		},
	}
	if !AnyEnabled(cfg) {
		t.Fatal("slack enabled should count")
	}
}

func TestDispatchTest_slackDefaultEvent(t *testing.T) {
	var got slackWebhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Config{
		Notify: config.NotifyCfg{
			Slack: config.WebhookCfg{Enabled: true, WebhookURL: srv.URL},
			Retry: config.NotifyRetryCfg{MaxAttempts: 1},
		},
	}
	if err := DispatchTest(context.Background(), cfg, EventTest); err != nil {
		t.Fatal(err)
	}
	if got.Text != "" {
		t.Fatalf("top-level text must be empty, got %q", got.Text)
	}
	if len(got.Attachments) != 1 {
		t.Fatalf("attachments=%d", len(got.Attachments))
	}
	att := got.Attachments[0]
	if !strings.Contains(att.Fallback, "GROOT notify test") {
		t.Fatalf("fallback=%q", att.Fallback)
	}
	if !strings.Contains(att.Text, "GROOT notify test") {
		t.Fatalf("attachment text=%q", att.Text)
	}
}

func TestDispatchTest_failureEvent(t *testing.T) {
	var got slackWebhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Config{
		Notify: config.NotifyCfg{
			Slack: config.WebhookCfg{Enabled: true, WebhookURL: srv.URL},
			Retry: config.NotifyRetryCfg{MaxAttempts: 1},
		},
	}
	if err := DispatchTest(context.Background(), cfg, eventFailure); err != nil {
		t.Fatal(err)
	}
	if got.Text != "" {
		t.Fatalf("top-level text must be empty, got %q", got.Text)
	}
	if len(got.Attachments) != 1 {
		t.Fatalf("attachments=%d", len(got.Attachments))
	}
	att := got.Attachments[0]
	if !strings.Contains(att.Fallback, "GROOT FAILED") {
		t.Fatalf("fallback=%q", att.Fallback)
	}
	if !strings.Contains(att.Text, "GROOT failed") {
		t.Fatalf("attachment text=%q", att.Text)
	}
}

func TestDispatchTest_noChannels(t *testing.T) {
	err := DispatchTest(context.Background(), config.Config{}, EventTest)
	if err == nil || !strings.Contains(err.Error(), "no notify channel") {
		t.Fatalf("err=%v", err)
	}
}

func TestBuildSummaryLine_notifyTest(t *testing.T) {
	line := buildSummaryLine(EventTest, testSummary(), "")
	if !strings.Contains(line, "GROOT notify test") || !strings.Contains(line, "notify-test") {
		t.Fatalf("line=%q", line)
	}
	if strings.Contains(line, "client=") {
		t.Fatalf("empty client_id must omit client= attr: %q", line)
	}
}

func TestBuildSummaryLine_clientID(t *testing.T) {
	sum := testSummary()
	sum.ClientID = "bastion-01"
	line := buildSummaryLine(eventSuccess, sum, "")
	if !strings.Contains(line, "GROOT finished. client=bastion-01 total=") {
		t.Fatalf("line=%q", line)
	}
	failLine := buildSummaryLine(eventFailure, sum, "boom")
	if !strings.Contains(failLine, "GROOT FAILED. client=bastion-01 reason=boom") {
		t.Fatalf("failLine=%q", failLine)
	}
	partial := buildSummaryLine(eventFailure, sum, "")
	if !strings.Contains(partial, "GROOT finished with failures. client=bastion-01 total=") {
		t.Fatalf("partial=%q", partial)
	}
	testLine := buildSummaryLine(EventTest, sum, "")
	if !strings.Contains(testLine, "GROOT notify test. client=bastion-01 total=") {
		t.Fatalf("testLine=%q", testLine)
	}
}
