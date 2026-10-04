package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

// devBypassToken はフロントがサイトキー未設定のときに送る固定値（ReservationForm.tsx）
const devBypassToken = "dev-bypass"

// siteverify の error-codes のうち、秘密鍵の設定に起因するもの
// 秘密鍵はこちらだけが持つため外部からは起こせず、全員の予約が失敗するので ERROR（メール通知）にする
var misconfiguredErrorCodes = map[string]bool{
	"missing-input-secret": true,
	"invalid-input-secret": true,
}

// siteverify の error-codes のうち、調査は要るがメール通知にはしないもの
// internal-error は Cloudflare 側の一時障害。bad-request はリクエスト不正で、
// 送られたトークンの内容でも起こりうる（外部から鳴らせる）ため ERROR にしない
var abnormalErrorCodes = map[string]bool{
	"internal-error": true,
	"bad-request":    true,
}

// Verify は Turnstile トークンを検証する（remoteip は Cloud Run 等で不一致になり得るため送らない）
// 検証失敗はすべて ErrCaptchaFailed（400）で返すが、原因の区別はログのレベルで残す（docs/monitoring.md §4.6 L1）
func (v *Verifier) Verify(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.ErrCaptchaFailed
	}
	if token == devBypassToken {
		// この検証器は development 以外でしか使わない。本番ビルドのフロントはサイトキー未設定なら送信自体を止めるため、
		// ここに来るのはローカル開発から staging を叩いたときか、外部で作られたリクエスト。
		// 誰でも送れる値なので ERROR（メール通知）にはしない
		slog.Warn("turnstile dev-bypass token received")
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
		logRejection(ctx, result.ErrorCodes)
		return domain.ErrCaptchaFailed
	}
	return nil
}

// rejectionLevel は siteverify の error-codes から記録レベルを決める
// 秘密鍵の設定起因 → ERROR、Cloudflare 障害・リクエスト不正 → WARN、それ以外（利用者起因）→ INFO
func rejectionLevel(codes []string) slog.Level {
	level := slog.LevelInfo
	for _, code := range codes {
		if misconfiguredErrorCodes[code] {
			return slog.LevelError
		}
		if abnormalErrorCodes[code] {
			level = slog.LevelWarn
		}
	}
	return level
}

func logRejection(ctx context.Context, codes []string) {
	level := rejectionLevel(codes)
	msg := "turnstile verification rejected"
	switch level {
	case slog.LevelError:
		msg = "turnstile verification misconfigured"
	case slog.LevelWarn:
		msg = "turnstile verification abnormal"
	}
	slog.Log(ctx, level, msg, "error_codes", codes)
}
