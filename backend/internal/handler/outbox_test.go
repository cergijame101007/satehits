package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
	inframail "github.com/cergijame101007/satehits/internal/infrastructure/mail"
)

type emptyOutboxRepo struct{}

func (emptyOutboxRepo) Enqueue(context.Context, domain.EnqueueEmailInput) error { return nil }
func (emptyOutboxRepo) ClaimNextPending(context.Context, time.Time) (*domain.EmailOutboxMessage, error) {
	return nil, nil
}
func (emptyOutboxRepo) MarkSent(context.Context, uuid.UUID) error { return nil }
func (emptyOutboxRepo) MarkRetry(context.Context, uuid.UUID, time.Time, string) error {
	return nil
}
func (emptyOutboxRepo) MarkFailed(context.Context, uuid.UUID, string) error { return nil }
func (emptyOutboxRepo) ReleaseClaim(context.Context, uuid.UUID, time.Time, string) error {
	return nil
}

type noopMailSender struct{}

func (noopMailSender) Send(context.Context, domain.MailMessage) error { return nil }

func TestOutboxHandlerFlushAcceptsPost(t *testing.T) {
	d := inframail.NewDispatcher(emptyOutboxRepo{}, noopMailSender{}, inframail.Config{BatchSize: 5})
	h := NewOutboxHandler(d)

	req := httptest.NewRequest(http.MethodPost, "/internal/outbox/flush", nil)
	rr := httptest.NewRecorder()
	h.HandleFlush(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var stats inframail.ProcessStats
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("json: %v", err)
	}
	if stats.Processed != 0 {
		t.Fatalf("processed = %d, want 0", stats.Processed)
	}
}

func TestOutboxHandlerFlushRejectsGet(t *testing.T) {
	d := inframail.NewDispatcher(emptyOutboxRepo{}, noopMailSender{}, inframail.Config{BatchSize: 5})
	h := NewOutboxHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/internal/outbox/flush", nil)
	rr := httptest.NewRecorder()
	h.HandleFlush(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}
