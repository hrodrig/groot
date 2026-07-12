package notifier

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
)

const (
	eventSuccess = "success"
	eventFailure = "failure"
)

// telegramAPIBase is the Bot API origin (overridable in tests).
var telegramAPIBase = "https://api.telegram.org"

// FanOut dispatches notifications to enabled channels.
type FanOut struct {
	senders   []Sender
	pagerDuty []pagerDutyRoute
	onFailure config.OnFailureCfg
	http      *httpNotify
}

// Sender is a notification target.
type Sender interface {
	Send(ctx context.Context, text string) error
}

// NewFanOut creates the enabled notifiers.
func NewFanOut(cfg config.Config) *FanOut {
	httpN := newHTTPNotify(retryConfig{
		maxAttempts:    cfg.Notify.Retry.MaxAttempts,
		initialBackoff: cfg.Notify.Retry.InitialBackoff,
		maxBackoff:     cfg.Notify.Retry.MaxBackoff,
	})

	n := cfg.Notify
	senders := make([]Sender, 0, 8)
	senders = appendWebhookSenders(senders, httpN, n.Slack.Enabled, n.Slack.WebhookURL, "slack")
	senders = appendWebhookSenders(senders, httpN, n.Discord.Enabled, n.Discord.WebhookURL, "discord")
	senders = appendWebhookSenders(senders, httpN, n.Teams.Enabled, n.Teams.WebhookURL, "teams")
	senders = appendTelegramSenders(senders, httpN, n.Telegram)
	senders = appendGenericSenders(senders, httpN, n.Generic)
	senders = appendEmailSenders(senders, emailCfgView{
		enabled:    n.Email.Enabled,
		host:       n.Email.Host,
		port:       n.Email.Port,
		username:   n.Email.Username,
		password:   n.Email.Password,
		from:       n.Email.From,
		to:         n.Email.To,
		useTLS:     n.Email.UseTLS,
		skipVerify: n.Email.SkipVerify,
	})
	return &FanOut{
		senders:   senders,
		pagerDuty: pagerDutyRoutesFrom(n.PagerDuty),
		onFailure: n.OnFailure,
		http:      httpN,
	}
}

// Destinations returns the number of configured notify targets (senders + PagerDuty routes).
func (f *FanOut) Destinations() int {
	return len(f.senders) + len(f.pagerDuty)
}

// ShouldNotifyPartialFailure reports whether a completed collect should emit a failure alert.
func ShouldNotifyPartialFailure(cfg config.Config, summary collector.Summary) bool {
	of := cfg.Notify.OnFailure
	return of.Enabled && summary.Failed >= of.MinFailedJobs
}

// ShouldNotifyAbort reports whether an aborted collect should emit a failure alert.
func ShouldNotifyAbort(cfg config.Config) bool {
	of := cfg.Notify.OnFailure
	return of.Enabled && of.OnAbort
}

func appendWebhookSenders(out []Sender, httpN *httpNotify, enabled bool, rawURLs, kind string) []Sender {
	if !enabled {
		return out
	}
	for _, u := range config.SplitSemicolonList(rawURLs) {
		out = append(out, &webhookSender{url: u, kind: kind, http: httpN})
	}
	return out
}

func appendTelegramSenders(out []Sender, httpN *httpNotify, t config.TelegramCfg) []Sender {
	if !t.Enabled || strings.TrimSpace(t.Token) == "" {
		return out
	}
	for _, chatID := range config.SplitSemicolonList(t.ChatID) {
		out = append(out, &telegramSender{token: t.Token, chatID: chatID, http: httpN})
	}
	return out
}

func appendGenericSenders(out []Sender, httpN *httpNotify, g config.GenericWebhookCfg) []Sender {
	if !g.Enabled {
		return out
	}
	key := strings.TrimSpace(g.JSONKey)
	if key == "" {
		key = "text"
	}
	hdr := cloneStringMap(g.Headers)
	extra := cloneStringMap(g.ExtraFields)
	hmacHeader := strings.TrimSpace(g.HMACHeader)
	if hmacHeader == "" {
		hmacHeader = "X-Groot-Signature"
	}
	for _, u := range config.SplitSemicolonList(g.WebhookURL) {
		out = append(out, &genericWebhookSender{
			url:          u,
			jsonKey:      key,
			headers:      hdr,
			extraFields:  extra,
			bodyTemplate: strings.TrimSpace(g.BodyTemplate),
			hmacSecret:   g.HMACSecret,
			hmacHeader:   hmacHeader,
			http:         httpN,
		})
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

// Notify sends the collection summary to all enabled destinations (success event).
func (f *FanOut) Notify(ctx context.Context, summary collector.Summary) error {
	return f.notifyEvent(ctx, eventSuccess, summary, "")
}

// NotifyFailure sends a failure alert to all enabled destinations.
func (f *FanOut) NotifyFailure(ctx context.Context, summary collector.Summary, reason string) error {
	return f.notifyEvent(ctx, eventFailure, summary, reason)
}

func (f *FanOut) notifyEvent(ctx context.Context, event string, summary collector.Summary, reason string) error {
	if len(f.senders) == 0 && len(f.pagerDuty) == 0 {
		return nil
	}

	msgCtx := messageContext{
		Event:       event,
		Summary:     summary,
		Reason:      reason,
		SummaryLine: buildSummaryLine(event, summary, reason),
	}

	var errs []error
	for _, sender := range f.senders {
		text := msgCtx.SummaryLine
		if gw, ok := sender.(*genericWebhookSender); ok {
			if err := gw.SendContext(ctx, msgCtx); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := sender.Send(ctx, text); err != nil {
			errs = append(errs, err)
		}
	}
	for _, route := range f.pagerDuty {
		if err := sendPagerDutyV2(ctx, f.http, route, msgCtx.SummaryLine, summary); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type webhookSender struct {
	url  string
	kind string
	http *httpNotify
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
	return w.http.postJSONWithRetry(ctx, w.url, body, nil)
}

type genericWebhookSender struct {
	url          string
	jsonKey      string
	headers      map[string]string
	extraFields  map[string]string
	bodyTemplate string
	hmacSecret   string
	hmacHeader   string
	http         *httpNotify
}

func (w *genericWebhookSender) Send(ctx context.Context, text string) error {
	return w.SendContext(ctx, messageContext{Event: eventSuccess, SummaryLine: text})
}

func (w *genericWebhookSender) SendContext(ctx context.Context, msgCtx messageContext) error {
	body, err := w.buildBody(msgCtx)
	if err != nil {
		return err
	}
	return w.http.postJSONWithRetry(ctx, w.url, body, func(req *http.Request) {
		for k, v := range w.headers {
			if strings.TrimSpace(k) != "" {
				req.Header.Set(k, v)
			}
		}
		if strings.TrimSpace(w.hmacSecret) != "" {
			mac := hmac.New(sha256.New, []byte(w.hmacSecret))
			_, _ = mac.Write(body)
			sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
			req.Header.Set(w.hmacHeader, sig)
		}
	})
}

func (w *genericWebhookSender) buildBody(msgCtx messageContext) ([]byte, error) {
	if w.bodyTemplate != "" {
		rendered := applyTemplate(w.bodyTemplate, msgCtx)
		if !json.Valid([]byte(rendered)) {
			return nil, fmt.Errorf("generic webhook body_template is not valid JSON after substitution")
		}
		return []byte(rendered), nil
	}

	payload := map[string]string{w.jsonKey: msgCtx.SummaryLine}
	for k, v := range w.extraFields {
		if strings.TrimSpace(k) == "" {
			continue
		}
		payload[k] = applyTemplate(v, msgCtx)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal generic webhook payload: %w", err)
	}
	return body, nil
}

type telegramSender struct {
	token  string
	chatID string
	http   *httpNotify
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

	resp, err := t.http.client.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram status %d", resp.StatusCode)
	}
	return nil
}
