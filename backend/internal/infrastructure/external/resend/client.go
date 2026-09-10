package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

const emailsURL = "https://api.resend.com/emails"

const defaultHTTPTimeout = 30 * time.Second

// Client は Resend の /emails API クライアント
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient は Resend クライアントを生成する
func NewClient(apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &Client{apiKey: apiKey, httpClient: httpClient}
}

type sendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// Send は domain.MailSender を実装する
func (c *Client) Send(ctx context.Context, msg domain.MailMessage) error {
	body, err := json.Marshal(sendEmailRequest{
		From:    msg.From,
		To:      []string{msg.To},
		Subject: msg.Subject,
		HTML:    msg.HTML,
		Text:    msg.Text,
	})
	if err != nil {
		return fmt.Errorf("resend marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, emailsURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if msg.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", msg.IdempotencyKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("resend status %d: read body: %w", resp.StatusCode, readErr)
	}
	bodyText := strings.TrimSpace(string(respBody))
	return classifyResendStatus(resp.StatusCode, bodyText)
}

// classifyResendStatus は HTTP ステータスと本文から再送可否を分類する。
// 5xx / 429 / 409 concurrent_idempotent_requests → 一時失敗（再送）。
// 401 / 403 → ErrMailAuth（設定起因。行の試行を消費せずバッチ中断）。
// それ以外の 4xx（invalid_idempotent_request 含む）→ ErrMailPermanent。
func classifyResendStatus(statusCode int, bodyText string) error {
	base := fmt.Errorf("resend status %d: %s", statusCode, bodyText)
	if statusCode >= 500 || statusCode == http.StatusTooManyRequests {
		return base
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return fmt.Errorf("%w: %w", domain.ErrMailAuth, base)
	}
	if statusCode == http.StatusConflict && strings.Contains(bodyText, "concurrent_idempotent_requests") {
		return base
	}
	if statusCode >= 400 && statusCode < 500 {
		return fmt.Errorf("%w: %w", domain.ErrMailPermanent, base)
	}
	return base
}
