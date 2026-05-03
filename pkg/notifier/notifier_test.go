package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/groot/pkg/collector"
	"github.com/hrodrig/groot/pkg/config"
)

func TestNewFanOut_empty(t *testing.T) {
	f := NewFanOut(config.Config{})
	if f == nil || len(f.senders) != 0 || len(f.pagerDuty) != 0 {
		t.Fatalf("expected no senders, got senders=%d pagerduty=%d", len(f.senders), len(f.pagerDuty))
	}
}

func TestFanOut_Notify_noSenders(t *testing.T) {
	f := &FanOut{}
	if err := f.Notify(context.Background(), collector.Summary{Total: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestDiscordWebhookContent_truncates(t *testing.T) {
	long := strings.Repeat("x", 2100)
	got := discordWebhookContent(long)
	if len([]rune(got)) > 2000 {
		t.Fatalf("len=%d", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("suffix: %q", got)
	}
}

func TestWebhookSender_discordOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["content"] == "" {
			t.Fatal("expected content field")
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	w := &webhookSender{url: srv.URL, kind: "discord"}
	if err := w.Send(context.Background(), "hello"); err != nil {
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

func TestFanOut_Notify_slackMultipleWebhooks(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Config{}
	cfg.Notify.Slack.Enabled = true
	cfg.Notify.Slack.WebhookURL = srv.URL + " ; " + srv.URL
	f := NewFanOut(cfg)
	if len(f.senders) != 2 {
		t.Fatalf("expected 2 slack senders, got %d", len(f.senders))
	}
	summary := collector.Summary{
		Total:       1,
		Success:     1,
		Failed:      0,
		Duration:    time.Second,
		OutputDir:   "/tmp/out",
		ArchivePath: "/tmp/a.tgz",
	}
	if err := f.Notify(context.Background(), summary); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("expected 2 POSTs, got %d", hits)
	}
}

func TestFanOut_Notify_telegramMultipleChatIDs(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("chat_id") == "" {
			t.Fatal("expected chat_id")
		}
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	token := "TESTTOKEN"
	orig := telegramAPIBase
	telegramAPIBase = srv.URL
	t.Cleanup(func() { telegramAPIBase = orig })

	cfg := config.Config{}
	cfg.Notify.Telegram.Enabled = true
	cfg.Notify.Telegram.Token = token
	cfg.Notify.Telegram.ChatID = "111;222"
	f := NewFanOut(cfg)
	if len(f.senders) != 2 {
		t.Fatalf("expected 2 telegram senders, got %d", len(f.senders))
	}
	summary := collector.Summary{Total: 1, Success: 1, Duration: time.Second, OutputDir: "/o", ArchivePath: "/a.tgz"}
	if err := f.Notify(context.Background(), summary); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("expected 2 telegram POSTs, got %d", hits)
	}
}

func TestGenericWebhookSender_customJSONKeyAndHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "1" {
			t.Fatal("missing header")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["content"] != "hello" {
			t.Fatalf("body %#v", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	w := &genericWebhookSender{
		url:     srv.URL,
		jsonKey: "content",
		headers: map[string]string{"X-Test": "1"},
	}
	if err := w.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
}

func TestFanOut_Notify_genericMultipleURLs(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Config{}
	cfg.Notify.Generic.Enabled = true
	cfg.Notify.Generic.WebhookURL = srv.URL + ";" + srv.URL
	cfg.Notify.Generic.JSONKey = "message"
	f := NewFanOut(cfg)
	if len(f.senders) != 2 {
		t.Fatalf("expected 2 generic senders, got %d", len(f.senders))
	}
	summary := collector.Summary{Total: 1, Success: 1, Duration: time.Second, OutputDir: "/o", ArchivePath: "/a.tgz"}
	if err := f.Notify(context.Background(), summary); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("expected 2 POSTs, got %d", hits)
	}
}

func TestFanOut_Notify_pagerDuty(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	orig := pagerDutyEventsURL
	pagerDutyEventsURL = srv.URL
	t.Cleanup(func() { pagerDutyEventsURL = orig })

	cfg := config.Config{}
	cfg.Notify.PagerDuty.Enabled = true
	cfg.Notify.PagerDuty.RoutingKey = "rk-test"
	cfg.Notify.PagerDuty.Severity = "info"
	cfg.Notify.PagerDuty.Source = "unittest"

	f := NewFanOut(cfg)
	if len(f.pagerDuty) != 1 {
		t.Fatalf("pagerduty routes %d", len(f.pagerDuty))
	}
	sum := collector.Summary{Total: 3, Success: 2, Failed: 1, Duration: 2 * time.Second, OutputDir: "/out", ArchivePath: "/a.tgz"}
	if err := f.Notify(context.Background(), sum); err != nil {
		t.Fatal(err)
	}
	var env struct {
		RoutingKey  string `json:"routing_key"`
		EventAction string `json:"event_action"`
		Payload     struct {
			Summary       string `json:"summary"`
			Severity      string `json:"severity"`
			Source        string `json:"source"`
			CustomDetails struct {
				Total   float64 `json:"total"`
				Success float64 `json:"success"`
			} `json:"custom_details"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatal(err)
	}
	if env.RoutingKey != "rk-test" || env.EventAction != "trigger" {
		t.Fatalf("%+v", env)
	}
	if env.Payload.Severity != "info" || env.Payload.Source != "unittest" {
		t.Fatalf("payload %+v", env.Payload)
	}
	if env.Payload.CustomDetails.Total != 3 || env.Payload.CustomDetails.Success != 2 {
		t.Fatalf("details %+v", env.Payload.CustomDetails)
	}
}
