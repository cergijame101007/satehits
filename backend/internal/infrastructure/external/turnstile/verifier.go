package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cergijame101007/satehits/internal/domain"
)

const siteVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Verifier は Cloudflare Turnstile の siteverify API クライアント
type Verifier struct {
	secret     string
	httpClient *http.Client
}

// NewVerifier は Turnstile 検証クライアントを生成する
func NewVerifier(secret string, httpClient *http.Client) *Verifier {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Verifier{secret: secret, httpClient: httpClient}
}

type siteVerifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify は Turnstile トークンを検証する（remoteip は Cloud Run 等で不一致になり得るため送らない）
func (v *Verifier) Verify(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" || token == "dev-bypass" {
		return domain.ErrCaptchaFailed
	}

	data := url.Values{}
	data.Set("secret", v.secret)
	data.Set("response", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, siteVerifyURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("turnstile request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("turnstile verify: %w", err)
	}
	defer resp.Body.Close()

	var result siteVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("turnstile decode: %w", err)
	}
	if !result.Success {
		return domain.ErrCaptchaFailed
	}
	return nil
}
