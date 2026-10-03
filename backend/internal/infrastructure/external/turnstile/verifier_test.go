package turnstile

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

// roundTripFunc は siteverify への HTTP 呼び出しを差し替える
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newVerifierReturning は siteverify が body を返す Verifier と、呼ばれた回数のカウンタを返す
func newVerifierReturning(t *testing.T, body string) (*Verifier, *int) {
	t.Helper()
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
	return NewVerifier("secret", client), &calls
}

// captureSlog は slog の既定ロガーを JSON でバッファへ向け、テスト終了時に戻す
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// docs/monitoring.md §4.6 L1
func TestRejectionLevel(t *testing.T) {
	tests := []struct {
		name  string
		codes []string
		want  slog.Level
	}{
		{name: "treats invalid secret as misconfiguration", codes: []string{"invalid-input-secret"}, want: slog.LevelError},
		{name: "treats missing secret as misconfiguration", codes: []string{"missing-input-secret"}, want: slog.LevelError},
		// bad-request は送られたトークンの内容でも起こりうるため、メール通知（ERROR）にしない
		{name: "treats bad request as abnormal", codes: []string{"bad-request"}, want: slog.LevelWarn},
		{name: "treats cloudflare internal error as abnormal", codes: []string{"internal-error"}, want: slog.LevelWarn},
		{name: "prefers misconfiguration over abnormal", codes: []string{"internal-error", "invalid-input-secret"}, want: slog.LevelError},
		{name: "treats expired or reused token as user side", codes: []string{"timeout-or-duplicate"}, want: slog.LevelInfo},
		{name: "treats invalid response token as user side", codes: []string{"invalid-input-response"}, want: slog.LevelInfo},
		{name: "treats missing error codes as user side", codes: nil, want: slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rejectionLevel(tt.codes); got != tt.want {
				t.Errorf("rejectionLevel(%v): got %s, want %s", tt.codes, got, tt.want)
			}
		})
	}
}

func TestVerifier_Verify(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		body      string
		wantErr   error
		wantCalls int
		wantLog   string
		wantLevel string
		wantNoLog bool
	}{
		{
			name:      "accepts a successful verification without logging",
			token:     "valid-token",
			body:      `{"success":true}`,
			wantCalls: 1,
			wantNoLog: true,
		},
		{
			name:      "rejects an empty token without calling siteverify",
			token:     "  ",
			wantErr:   domain.ErrCaptchaFailed,
			wantCalls: 0,
			wantNoLog: true,
		},
		{
			name:      "logs a warning for the dev-bypass token without calling siteverify",
			token:     "dev-bypass",
			wantErr:   domain.ErrCaptchaFailed,
			wantCalls: 0,
			wantLog:   "turnstile dev-bypass token received",
			wantLevel: "WARN",
		},
		{
			name:      "logs an error when the secret is invalid",
			token:     "token",
			body:      `{"success":false,"error-codes":["invalid-input-secret"]}`,
			wantErr:   domain.ErrCaptchaFailed,
			wantCalls: 1,
			wantLog:   "turnstile verification misconfigured",
			wantLevel: "ERROR",
		},
		{
			name:      "logs a warning when cloudflare reports an internal error",
			token:     "token",
			body:      `{"success":false,"error-codes":["internal-error"]}`,
			wantErr:   domain.ErrCaptchaFailed,
			wantCalls: 1,
			wantLog:   "turnstile verification abnormal",
			wantLevel: "WARN",
		},
		{
			name:      "logs info when the user token is expired",
			token:     "token",
			body:      `{"success":false,"error-codes":["timeout-or-duplicate"]}`,
			wantErr:   domain.ErrCaptchaFailed,
			wantCalls: 1,
			wantLog:   "turnstile verification rejected",
			wantLevel: "INFO",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf := captureSlog(t)
			v, calls := newVerifierReturning(t, tt.body)

			err := v.Verify(t.Context(), tt.token)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err: got %v, want %v", err, tt.wantErr)
			}
			if *calls != tt.wantCalls {
				t.Errorf("siteverify calls: got %d, want %d", *calls, tt.wantCalls)
			}
			logText := logBuf.String()
			if tt.wantNoLog {
				if logText != "" {
					t.Errorf("log should be empty: %s", logText)
				}
				return
			}
			if !strings.Contains(logText, `"msg":"`+tt.wantLog+`"`) {
				t.Errorf("log should contain %q: %s", tt.wantLog, logText)
			}
			if !strings.Contains(logText, `"level":"`+tt.wantLevel+`"`) {
				t.Errorf("log level: want %s: %s", tt.wantLevel, logText)
			}
			// トークン値はログに出さない
			if strings.Contains(logText, `"`+tt.token+`"`) {
				t.Errorf("log should not contain the token: %s", logText)
			}
		})
	}
}
