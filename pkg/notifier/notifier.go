package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"groot/pkg/collector"
	"groot/pkg/config"
)

// telegramAPIBase is the Bot API origin (overridable in tests).
var telegramAPIBase = "https://api.telegram.org"

// notifyHTTPClient is used for all outbound HTTP notification requests (webhooks, Telegram, PagerDuty).
// A bounded timeout avoids hanging collect when a remote endpoint stalls.
var notifyHTTPClient = &http.Client{Timeout: 30 * time.Second}

// FanOut dispatches notifications to enabled channels.
type FanOut struct {
	senders   []Sender
	pagerDuty []pagerDutyRoute
}

// Sender is a notification target.
type Sender interface {
	Send(ctx context.Context, text string) error
}

// NewFanOut creates the enabled notifiers.
func NewFanOut(cfg config.Config) *FanOut {
	n := cfg.Notify
	senders := make([]Sender, 0, 8)
	senders = appendWebhookSenders(senders, n.Slack.Enabled, n.Slack.WebhookURL, "slack")
	senders = appendWebhookSenders(senders, n.Discord.Enabled, n.Discord.WebhookURL, "discord")
	senders = appendWebhookSenders(senders, n.Teams.Enabled, n.Teams.WebhookURL, "teams")
	senders = appendTelegramSenders(senders, n.Telegram)
	senders = appendGenericSenders(senders, n.Generic)
	return &FanOut{senders: senders, pagerDuty: pagerDutyRoutesFrom(n.PagerDuty)}
}

func appendWebhookSenders(out []Sender, enabled bool, rawURLs, kind string) []Sender {
	if !enabled {
		return out
	}
	for _, u := range config.SplitSemicolonList(rawURLs) {
		out = append(out, &webhookSender{url: u, kind: kind})
	}
	return out
}

func appendTelegramSenders(out []Sender, t config.TelegramCfg) []Sender {
	if !t.Enabled || strings.TrimSpace(t.Token) == "" {
		return out
	}
	for _, chatID := range config.SplitSemicolonList(t.ChatID) {
		out = append(out, &telegramSender{token: t.Token, chatID: chatID})
	}
	return out
}

func appendGenericSenders(out []Sender, g config.GenericWebhookCfg) []Sender {
	if !g.Enabled {
		return out
	}
	key := strings.TrimSpace(g.JSONKey)
	if key == "" {
		key = "text"
	}
	hdr := cloneStringMap(g.Headers)
	for _, u := range config.SplitSemicolonList(g.WebhookURL) {
		out = append(out, &genericWebhookSender{url: u, jsonKey: key, headers: hdr})
	}
	return out
}

func pagerDutyRoutesFrom(p config.PagerDutyCfg) []pagerDutyRoute {
	if !p.Enabled {
		return nil
	}
	out := make([]pagerDutyRoute, 0)
	for _, rk := range config.SplitSemicolonList(p.RoutingKey) {
		out = append(out, pagerDutyRoute{routingKey: rk, severity: p.Severity, source: p.Source})
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// discordWebhookContent truncates text to Discord's incoming-webhook "content" limit (2000 characters).
func discordWebhookContent(text string) string {
	const max = 2000
	r := []rune(text)
	if len(r) <= max {
		return text
	}
	const suffix = "..."
	sr := []rune(suffix)
	cut := max - len(sr)
	if cut < 0 {
		cut = 0
	}
	return string(r[:cut]) + suffix
}

// Notify sends the collection summary to all enabled destinations.
func (f *FanOut) Notify(ctx context.Context, summary collector.Summary) error {
	if len(f.senders) == 0 && len(f.pagerDuty) == 0 {
		return nil
	}

	text := fmt.Sprintf(
		"GROOT finished. total=%d success=%d failed=%d duration=%s output=%s archive=%s",
		summary.Total,
		summary.Success,
		summary.Failed,
		summary.Duration.Round(time.Second),
		summary.OutputDir,
		summary.ArchivePath,
	)

	var errs []error
	for _, sender := range f.senders {
		if err := sender.Send(ctx, text); err != nil {
			errs = append(errs, err)
		}
	}
	for _, route := range f.pagerDuty {
		if err := sendPagerDutyV2(ctx, route, text, summary); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type webhookSender struct {
	url  string
	kind string
}

func (w *webhookSender) Send(ctx context.Context, text string) error {
	var payload any
	switch w.kind {
	case "slack":
		payload = map[string]string{"text": text}
	case "teams":
		payload = map[string]string{"text": text}
	case "discord":
		payload = map[string]string{"content": discordWebhookContent(text)}
	default:
		payload = map[string]string{"message": text}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := notifyHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}

type genericWebhookSender struct {
	url     string
	jsonKey string
	headers map[string]string
}

func (w *genericWebhookSender) Send(ctx context.Context, text string) error {
	payload := map[string]string{w.jsonKey: text}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal generic webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create generic webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := notifyHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send generic webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("generic webhook status %d", resp.StatusCode)
	}
	return nil
}

type telegramSender struct {
	token  string
	chatID string
}

func (t *telegramSender) Send(ctx context.Context, text string) error {
	u := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPIBase, t.token)
	data := url.Values{}
	data.Set("chat_id", t.chatID)
	data.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := notifyHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram status %d", resp.StatusCode)
	}
	return nil
}
