package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groot/pkg/collector"
	"groot/pkg/config"
)

func TestNewFanOut_empty(t *testing.T) {
	f := NewFanOut(config.Config{})
	if f == nil || len(f.senders) != 0 {
		t.Fatalf("expected no senders, got %d", len(f.senders))
	}
}

func TestFanOut_Notify_noSenders(t *testing.T) {
	f := &FanOut{}
	if err := f.Notify(context.Background(), collector.Summary{Total: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookSender_slackOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["text"] == "" {
			t.Fatal("expected text field")
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	w := &webhookSender{url: srv.URL, kind: "slack"}
	if err := w.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookSender_unknownKindUsesMessageField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["message"] == "" {
			t.Fatalf("expected message field for unknown kind: %#v", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	w := &webhookSender{url: srv.URL, kind: "other"}
	if err := w.Send(context.Background(), "ping"); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookSender_badStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	w := &webhookSender{url: srv.URL, kind: "teams"}
	if err := w.Send(context.Background(), "x"); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestFanOut_Notify_slack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Config{}
	cfg.Notify.Slack.Enabled = true
	cfg.Notify.Slack.WebhookURL = srv.URL
	f := NewFanOut(cfg)
	summary := collector.Summary{
		Total:       2,
		Success:     2,
		Failed:      0,
		Duration:    time.Minute,
		OutputDir:   "/tmp/out",
		ArchivePath: "/tmp/a.tgz",
	}
	if err := f.Notify(context.Background(), summary); err != nil {
		t.Fatal(err)
	}
}
