package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"groot/pkg/collector"
	"groot/pkg/config"
)

// FanOut dispatches notifications to enabled channels.
type FanOut struct {
	senders []Sender
}

// Sender is a notification target.
type Sender interface {
	Send(ctx context.Context, text string) error
}

// NewFanOut creates the enabled notifiers.
func NewFanOut(cfg config.Config) *FanOut {
	senders := make([]Sender, 0, 3)
	if cfg.Notify.Slack.Enabled && cfg.Notify.Slack.WebhookURL != "" {
		senders = append(senders, &webhookSender{url: cfg.Notify.Slack.WebhookURL, kind: "slack"})
	}
	if cfg.Notify.Teams.Enabled && cfg.Notify.Teams.WebhookURL != "" {
		senders = append(senders, &webhookSender{url: cfg.Notify.Teams.WebhookURL, kind: "teams"})
	}
	if cfg.Notify.Telegram.Enabled && cfg.Notify.Telegram.Token != "" && cfg.Notify.Telegram.ChatID != "" {
		senders = append(senders, &telegramSender{token: cfg.Notify.Telegram.Token, chatID: cfg.Notify.Telegram.ChatID})
	}
	return &FanOut{senders: senders}
}

// Notify sends the collection summary to all enabled destinations.
func (f *FanOut) Notify(ctx context.Context, summary collector.Summary) error {
	if len(f.senders) == 0 {
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

	for _, sender := range f.senders {
		if err := sender.Send(ctx, text); err != nil {
			return err
		}
	}
	return nil
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}

type telegramSender struct {
	token  string
	chatID string
}

func (t *telegramSender) Send(ctx context.Context, text string) error {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	data := url.Values{}
	data.Set("chat_id", t.chatID)
	data.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram status %d", resp.StatusCode)
	}
	return nil
}
