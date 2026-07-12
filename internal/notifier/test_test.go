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
	var gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotText = body["text"]
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
	if !strings.Contains(gotText, "GROOT notify test") {
		t.Fatalf("text=%q", gotText)
	}
}

func TestDispatchTest_failureEvent(t *testing.T) {
	var gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotText = body["text"]
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
	if !strings.Contains(gotText, "GROOT FAILED") {
		t.Fatalf("text=%q", gotText)
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
}
