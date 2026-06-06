package notifier

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

func TestPostJSONWithRetry_retries5xx(t *testing.T) {
	applyRetryConfig(retryConfig{maxAttempts: 3, initialBackoff: time.Millisecond, maxBackoff: 5 * time.Millisecond})

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	if err := postJSONWithRetry(context.Background(), srv.URL, []byte(`{}`), nil); err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestPostJSONWithRetry_noRetry4xx(t *testing.T) {
	applyRetryConfig(retryConfig{maxAttempts: 3, initialBackoff: time.Millisecond, maxBackoff: 5 * time.Millisecond})

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	if err := postJSONWithRetry(context.Background(), srv.URL, []byte(`{}`), nil); err == nil {
		t.Fatal("expected error")
	}
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestGenericWebhookSender_bodyTemplateAndHMAC(t *testing.T) {
	secret := "test-secret"
	var gotBody []byte
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		gotSig = r.Header.Get("X-Groot-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	w := &genericWebhookSender{
		url:          srv.URL,
		bodyTemplate: `{"event":"{{event}}","failed":{{failed}},"text":"{{summary}}"}`,
		hmacSecret:   secret,
		hmacHeader:   "X-Groot-Signature",
	}
	sum := collector.Summary{Total: 2, Success: 1, Failed: 1, Duration: time.Second}
	msg := messageContext{Event: eventFailure, Summary: sum, SummaryLine: "line", Reason: "boom"}
	if err := w.SendContext(context.Background(), msg); err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["event"] != eventFailure || parsed["failed"] != float64(1) {
		t.Fatalf("%+v", parsed)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(gotBody)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if gotSig != want {
		t.Fatalf("sig=%q want=%q", gotSig, want)
	}
}

func TestGenericWebhookSender_extraFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["text"] == "" || body["source"] != "groot" {
			t.Fatalf("%+v", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	w := &genericWebhookSender{
		url:         srv.URL,
		jsonKey:     "text",
		extraFields: map[string]string{"source": "groot"},
	}
	if err := w.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
}

func TestNotifyFailure_message(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Config{}
	cfg.Notify.Slack.Enabled = true
	cfg.Notify.Slack.WebhookURL = srv.URL
	f := NewFanOut(cfg)
	sum := collector.Summary{Total: 1, Failed: 1, Duration: time.Second, OutputDir: "/o"}
	if err := f.NotifyFailure(context.Background(), sum, "archive failed"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "GROOT FAILED") || !strings.Contains(body, "archive failed") {
		t.Fatalf("body=%q", body)
	}
}

func TestShouldNotifyPartialFailure(t *testing.T) {
	cfg := config.Config{}
	cfg.Notify.OnFailure.Enabled = true
	cfg.Notify.OnFailure.MinFailedJobs = 2
	sum := collector.Summary{Failed: 1}
	if ShouldNotifyPartialFailure(cfg, sum) {
		t.Fatal("expected false below threshold")
	}
	sum.Failed = 2
	if !ShouldNotifyPartialFailure(cfg, sum) {
		t.Fatal("expected true at threshold")
	}
}

func TestShouldNotifyAbort(t *testing.T) {
	cfg := config.Config{}
	cfg.Notify.OnFailure.Enabled = true
	cfg.Notify.OnFailure.OnAbort = true
	if !ShouldNotifyAbort(cfg) {
		t.Fatal("expected true")
	}
	cfg.Notify.OnFailure.OnAbort = false
	if ShouldNotifyAbort(cfg) {
		t.Fatal("expected false when on_abort disabled")
	}
}

func TestApplyTemplate(t *testing.T) {
	out := applyTemplate(`{{event}} {{total}} {{reason}}`, messageContext{
		Event:   eventSuccess,
		Summary: collector.Summary{Total: 5},
		Reason:  "ok",
	})
	if out != "success 5 ok" {
		t.Fatalf("got %q", out)
	}
}
