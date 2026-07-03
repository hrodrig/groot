package notifier

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// retryConfig controls transient HTTP notify retries (5xx and network errors).
type retryConfig struct {
	maxAttempts    int
	initialBackoff time.Duration
	maxBackoff     time.Duration
}

func normalizeRetryConfig(cfg retryConfig) retryConfig {
	if cfg.maxAttempts < 1 {
		cfg.maxAttempts = 1
	}
	if cfg.initialBackoff <= 0 {
		cfg.initialBackoff = time.Second
	}
	if cfg.maxBackoff <= 0 {
		cfg.maxBackoff = 10 * time.Second
	}
	return cfg
}

// httpNotify performs outbound HTTP notification requests with retry.
type httpNotify struct {
	retry  retryConfig
	client *http.Client
}

func newHTTPNotify(cfg retryConfig) *httpNotify {
	return &httpNotify{
		retry:  normalizeRetryConfig(cfg),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *httpNotify) postJSONWithRetry(ctx context.Context, url string, body []byte, setHeaders func(*http.Request)) error {
	var lastErr error
	backoff := h.retry.initialBackoff

	for attempt := 1; attempt <= h.retry.maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if setHeaders != nil {
			setHeaders(req)
		}

		resp, err := h.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("send request: %w", err)
		} else {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode < 300 {
				return nil
			}
			if resp.StatusCode < 500 {
				return fmt.Errorf("HTTP status %d", resp.StatusCode)
			}
			lastErr = fmt.Errorf("HTTP status %d", resp.StatusCode)
		}

		if attempt == h.retry.maxAttempts {
			break
		}
		if !sleepCtx(ctx, backoff) {
			return lastErr
		}
		if backoff < h.retry.maxBackoff {
			backoff *= 2
			if backoff > h.retry.maxBackoff {
				backoff = h.retry.maxBackoff
			}
		}
	}
	return lastErr
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
