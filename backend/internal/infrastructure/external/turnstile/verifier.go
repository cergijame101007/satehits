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

// Verify は Turnstile トークンを検証する
func (v *Verifier) Verify(ctx context.Context, token, remoteIP string) error {
	if strings.TrimSpace(token) == "" {
		return domain.ErrCaptchaFailed
	}

	data := url.Values{}
	data.Set("secret", v.secret)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

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
