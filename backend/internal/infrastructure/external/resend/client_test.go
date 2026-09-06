package resend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestClientSendSetsIdempotencyKey(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"re_123"}`)
	}))
	defer srv.Close()

	c := NewClient("re_test", srv.Client())
	// Override URL via temporary helper: use custom transport by pointing client at srv.
	// Client hardcodes emailsURL; test classify + header via Send with rewritten RoundTrip.
	c.httpClient = &http.Client{Transport: rewriteHostTransport{base: srv.Client().Transport, host: srv.URL}}

	err := c.Send(context.Background(), domain.MailMessage{
		From:           "noreply@satehits.com",
		To:             "a@example.com",
		Subject:        "hi",
		Text:           "hi",
		IdempotencyKey: "reservation_received/550e8400-e29b-41d4-a716-446655440000",
	})
	if err != nil {
		t.Fatalf("Send() err = %v", err)
	}
	if gotKey != "reservation_received/550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("Idempotency-Key = %q", gotKey)
	}
}

type rewriteHostTransport struct {
	base http.RoundTripper
	host string
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u := *req.URL
	baseURL := strings.TrimPrefix(t.host, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")
	u.Scheme = "http"
	u.Host = baseURL
	req2 := req.Clone(req.Context())
	req2.URL = &u
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}

func TestClassifyResendStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantPerm   bool
	}{
		{name: "500 is transient", statusCode: 500, body: "oops", wantPerm: false},
		{name: "429 is transient", statusCode: 429, body: "rate", wantPerm: false},
		{name: "409 concurrent is transient", statusCode: 409, body: `{"name":"concurrent_idempotent_requests"}`, wantPerm: false},
		{name: "409 invalid idempotent is permanent", statusCode: 409, body: `{"name":"invalid_idempotent_request"}`, wantPerm: true},
		{name: "400 is permanent", statusCode: 400, body: "bad", wantPerm: true},
		{name: "422 is permanent", statusCode: 422, body: "invalid", wantPerm: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyResendStatus(tt.statusCode, tt.body)
			if err == nil {
				t.Fatal("err = nil")
			}
			isPerm := errors.Is(err, domain.ErrMailPermanent)
			if isPerm != tt.wantPerm {
				t.Fatalf("permanent = %v, want %v; err=%v", isPerm, tt.wantPerm, err)
			}
		})
	}
}
