package mail

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/application"
	"github.com/cergijame101007/satehits/internal/domain"
)

type passThroughTxManager struct{}

func (passThroughTxManager) DoInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

var _ application.TxManager = passThroughTxManager{}

type fakeOutboxRepo struct {
	pending      []*domain.EmailOutboxMessage
	claimCalls   int
	sentIDs      []uuid.UUID
	retryCalls   []retryCall
	failedCalls  []failedCall
	claimErr     error
	markSentErr  error
	markRetryErr error
	markFailErr  error
}

type retryCall struct {
	id           uuid.UUID
	attemptCount int
	next         time.Time
	lastError    string
}

type failedCall struct {
	id           uuid.UUID
	attemptCount int
	lastError    string
}

func (f *fakeOutboxRepo) Enqueue(context.Context, domain.EnqueueEmailInput) error { return nil }

func (f *fakeOutboxRepo) ClaimNextPending(context.Context) (*domain.EmailOutboxMessage, error) {
	f.claimCalls++
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	if len(f.pending) == 0 {
		return nil, nil
	}
	msg := f.pending[0]
	f.pending = f.pending[1:]
	return msg, nil
}

func (f *fakeOutboxRepo) MarkSent(_ context.Context, id uuid.UUID) error {
	if f.markSentErr != nil {
		return f.markSentErr
	}
	f.sentIDs = append(f.sentIDs, id)
	return nil
}

func (f *fakeOutboxRepo) MarkRetry(_ context.Context, id uuid.UUID, attemptCount int, nextAttemptAt time.Time, lastError string) error {
	if f.markRetryErr != nil {
		return f.markRetryErr
	}
	f.retryCalls = append(f.retryCalls, retryCall{id: id, attemptCount: attemptCount, next: nextAttemptAt, lastError: lastError})
	return nil
}

func (f *fakeOutboxRepo) MarkFailed(_ context.Context, id uuid.UUID, attemptCount int, lastError string) error {
	if f.markFailErr != nil {
		return f.markFailErr
	}
	f.failedCalls = append(f.failedCalls, failedCall{id: id, attemptCount: attemptCount, lastError: lastError})
	return nil
}

type recordingSender struct {
	msgs []domain.MailMessage
	err  error
}

func (r *recordingSender) Send(_ context.Context, msg domain.MailMessage) error {
	r.msgs = append(r.msgs, msg)
	return r.err
}

func sampleOutbox(attempt int) *domain.EmailOutboxMessage {
	return &domain.EmailOutboxMessage{
		ID:            uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ReservationID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		MailType:      domain.MailTypeReservationReceived,
		FromAddress:   "noreply@satehits.com",
		ToAddress:     "a@example.com",
		Subject:       "test",
		BodyHTML:      "<p>hi</p>",
		BodyText:      "hi",
		AttemptCount:  attempt,
		Status:        "pending",
	}
}

func TestDispatcherMarksSentOnSuccess(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(0)}}
	sender := &recordingSender{}
	d := NewDispatcher(repo, sender, passThroughTxManager{}, 10)
	fixed := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	d.now = func() time.Time { return fixed }

	stats, err := d.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending() err = %v", err)
	}
	if stats.Sent != 1 || stats.Processed != 1 {
		t.Fatalf("stats = %+v, want sent=1 processed=1", stats)
	}
	if len(sender.msgs) != 1 {
		t.Fatalf("sender calls = %d, want 1", len(sender.msgs))
	}
	wantKey := "reservation_received/550e8400-e29b-41d4-a716-446655440000"
	if sender.msgs[0].IdempotencyKey != wantKey {
		t.Fatalf("IdempotencyKey = %q, want %q", sender.msgs[0].IdempotencyKey, wantKey)
	}
	if len(repo.sentIDs) != 1 {
		t.Fatalf("MarkSent calls = %d, want 1", len(repo.sentIDs))
	}
}

func TestDispatcherRetriesOnTransientError(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(0)}}
	sender := &recordingSender{err: errors.New("temporary")}
	d := NewDispatcher(repo, sender, passThroughTxManager{}, 10)
	fixed := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	d.now = func() time.Time { return fixed }

	stats, err := d.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending() err = %v", err)
	}
	if stats.Retried != 1 || stats.Failed != 0 {
		t.Fatalf("stats = %+v, want retried=1 failed=0", stats)
	}
	if len(repo.retryCalls) != 1 {
		t.Fatalf("retry calls = %d, want 1", len(repo.retryCalls))
	}
	rc := repo.retryCalls[0]
	if rc.attemptCount != 1 {
		t.Fatalf("attemptCount = %d, want 1", rc.attemptCount)
	}
	wantNext := fixed.Add(1 * time.Minute)
	if !rc.next.Equal(wantNext) {
		t.Fatalf("next = %v, want %v", rc.next, wantNext)
	}
}

func TestDispatcherMarksFailedOnPermanentError(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(0)}}
	sender := &recordingSender{err: domain.ErrMailPermanent}
	d := NewDispatcher(repo, sender, passThroughTxManager{}, 10)

	stats, err := d.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending() err = %v", err)
	}
	if stats.Failed != 1 || stats.Retried != 0 {
		t.Fatalf("stats = %+v, want failed=1 retried=0", stats)
	}
	if len(repo.failedCalls) != 1 {
		t.Fatalf("failed calls = %d, want 1", len(repo.failedCalls))
	}
	if repo.failedCalls[0].attemptCount != 1 {
		t.Fatalf("attemptCount = %d, want 1", repo.failedCalls[0].attemptCount)
	}
}

func TestDispatcherMarksFailedWhenAttemptsExhausted(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(5)}}
	sender := &recordingSender{err: errors.New("still failing")}
	d := NewDispatcher(repo, sender, passThroughTxManager{}, 10)

	stats, err := d.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending() err = %v", err)
	}
	if stats.Failed != 1 {
		t.Fatalf("stats.Failed = %d, want 1", stats.Failed)
	}
	if len(repo.failedCalls) != 1 || repo.failedCalls[0].attemptCount != 6 {
		t.Fatalf("failedCalls = %+v, want attemptCount=6", repo.failedCalls)
	}
}

func TestDispatcherStopsWhenNoPending(t *testing.T) {
	repo := &fakeOutboxRepo{}
	d := NewDispatcher(repo, &recordingSender{}, passThroughTxManager{}, 10)

	stats, err := d.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending() err = %v", err)
	}
	if stats.Processed != 0 {
		t.Fatalf("processed = %d, want 0", stats.Processed)
	}
	if repo.claimCalls != 1 {
		t.Fatalf("claimCalls = %d, want 1", repo.claimCalls)
	}
}
