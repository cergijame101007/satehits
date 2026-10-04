package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakePinger は Pinger のフェイク。block が true なら ctx が終わるまで待って ctx.Err() を返す
type fakePinger struct {
	err   error
	block bool
}

func (p fakePinger) PingContext(ctx context.Context) error {
	if p.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return p.err
}

// docs/monitoring.md §4.4
func TestHealthHandler_HandleHealth(t *testing.T) {
	tests := []struct {
		name         string
		pinger       fakePinger
		wantStatus   int
		wantBody     healthResponse
		wantNoStore  bool
		shortTimeout bool
	}{
		{
			name:        "returns 200 when the database responds",
			pinger:      fakePinger{},
			wantStatus:  http.StatusOK,
			wantBody:    healthResponse{Status: "ok", Database: "ok"},
			wantNoStore: true,
		},
		{
			name:        "returns 503 when the database ping fails",
			pinger:      fakePinger{err: errors.New("connection refused")},
			wantStatus:  http.StatusServiceUnavailable,
			wantBody:    healthResponse{Status: "error", Database: "error"},
			wantNoStore: true,
		},
		{
			name:         "returns 503 when the database ping times out",
			pinger:       fakePinger{block: true},
			wantStatus:   http.StatusServiceUnavailable,
			wantBody:     healthResponse{Status: "error", Database: "error"},
			wantNoStore:  true,
			shortTimeout: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureLog(t)
			h := NewHealthHandler(tt.pinger)
			if tt.shortTimeout {
				h.timeout = 10 * time.Millisecond
			}
			rr := httptest.NewRecorder()

			h.HandleHealth(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))

			if rr.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", rr.Code, tt.wantStatus)
			}
			var got healthResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode body: %v: %s", err, rr.Body.String())
			}
			if got != tt.wantBody {
				t.Errorf("body: got %+v, want %+v", got, tt.wantBody)
			}
			if tt.wantNoStore && rr.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("Cache-Control: got %q, want %q", rr.Header().Get("Cache-Control"), "no-store")
			}
		})
	}
}

func TestNewHealthHandler_usesTwoSecondTimeout(t *testing.T) {
	h := NewHealthHandler(fakePinger{})
	if h.timeout != 2*time.Second {
		t.Errorf("timeout: got %s, want %s", h.timeout, 2*time.Second)
	}
}
