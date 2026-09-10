package mail

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
)

type fakeOutboxRepo struct {
	pending      []*domain.EmailOutboxMessage
	claimCalls   int
	leaseUntils  []time.Time
	sentIDs      []uuid.UUID
	retryCalls   []retryCall
	failedCalls  []failedCall
	releaseCalls []releaseCall
	claimErr     error
	markSentErr  error
	markRetryErr error
	markFailErr  error
	releaseErr   error
}

type retryCall struct {
	id        uuid.UUID
	next      time.Time
	lastError string
}

type failedCall struct {
	id        uuid.UUID
	lastError string
}

type releaseCall struct {
	id        uuid.UUID
	next      time.Time
	lastError string
}

func (f *fakeOutboxRepo) Enqueue(context.Context, domain.EnqueueEmailInput) error { return nil }

// ClaimNextPending は本物と同じく attempt_count を +1 して返す
func (f *fakeOutboxRepo) ClaimNextPending(_ context.Context, leaseUntil time.Time) (*domain.EmailOutboxMessage, error) {
	f.claimCalls++
	f.leaseUntils = append(f.leaseUntils, leaseUntil)
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	if len(f.pending) == 0 {
		return nil, nil
	}
	msg := f.pending[0]
	f.pending = f.pending[1:]
	msg.AttemptCount++
	msg.NextAttemptAt = leaseUntil
	return msg, nil
}

func (f *fakeOutboxRepo) MarkSent(_ context.Context, id uuid.UUID) error {
	if f.markSentErr != nil {
		return f.markSentErr
	}
	f.sentIDs = append(f.sentIDs, id)
	return nil
}

func (f *fakeOutboxRepo) MarkRetry(_ context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error {
	if f.markRetryErr != nil {
		return f.markRetryErr
	}
	f.retryCalls = append(f.retryCalls, retryCall{id: id, next: nextAttemptAt, lastError: lastError})
	return nil
}

func (f *fakeOutboxRepo) MarkFailed(_ context.Context, id uuid.UUID, lastError string) error {
	if f.markFailErr != nil {
		return f.markFailErr
	}
	f.failedCalls = append(f.failedCalls, failedCall{id: id, lastError: lastError})
	return nil
}

func (f *fakeOutboxRepo) ReleaseClaim(_ context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error {
	if f.releaseErr != nil {
		return f.releaseErr
	}
	f.releaseCalls = append(f.releaseCalls, releaseCall{id: id, next: nextAttemptAt, lastError: lastError})
	return nil
}

type recordingSender struct {
	msgs        []domain.MailMessage
	hadDeadline []bool
	errs        []error // 呼び出し順に返す。足りなければ最後の要素を繰り返す
	onSend      func()
}

func (r *recordingSender) Send(ctx context.Context, msg domain.MailMessage) error {
	r.msgs = append(r.msgs, msg)
	_, ok := ctx.Deadline()
	r.hadDeadline = append(r.hadDeadline, ok)
	if r.onSend != nil {
		r.onSend()
	}
	if len(r.errs) == 0 {
		return nil
	}
	i := len(r.msgs) - 1
	if i >= len(r.errs) {
		i = len(r.errs) - 1
	}
	return r.errs[i]
}

var (
	outboxID1     = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	outboxID2     = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	reservationID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	fixedNow      = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
)

// sampleOutbox は claim 前の行（attemptCount は既に失敗した回数）
func sampleOutbox(id uuid.UUID, attemptsSoFar int) *domain.EmailOutboxMessage {
	return &domain.EmailOutboxMessage{
		ID:            id,
		ReservationID: reservationID,
		MailType:      domain.MailTypeReservationReceived,
		FromAddress:   "noreply@satehits.com",
		ToAddress:     "a@example.com",
		Subject:       "test",
		BodyHTML:      "<p>hi</p>",
		BodyText:      "hi",
		AttemptCount:  attemptsSoFar,
		Status:        domain.OutboxStatusPending,
	}
}

func newTestDispatcher(repo *fakeOutboxRepo, sender *recordingSender, cfg Config) *Dispatcher {
	d := NewDispatcher(repo, sender, cfg)
	d.now = func() time.Time { return fixedNow }
	return d
}

func TestDispatcherProcessPending(t *testing.T) {
	tests := []struct {
		name   string
		repo   *fakeOutboxRepo
		sender *recordingSender
		cfg    Config
		assert func(t *testing.T, repo *fakeOutboxRepo, sender *recordingSender, stats ProcessStats)
	}{
		{
			name:   "marks sent with idempotency key and lease on success",
			repo:   &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(outboxID1, 0)}},
			sender: &recordingSender{},
			cfg:    Config{BatchSize: 10},
			assert: func(t *testing.T, repo *fakeOutboxRepo, sender *recordingSender, stats ProcessStats) {
				t.Helper()
				if stats.Sent != 1 || stats.Processed != 1 || stats.Halted != "" {
					t.Fatalf("stats = %+v, want sent=1 processed=1 no halt", stats)
				}
				wantKey := "reservation_received/" + reservationID.String()
				if len(sender.msgs) != 1 || sender.msgs[0].IdempotencyKey != wantKey {
					t.Fatalf("sender msgs = %+v, want 1 with key %q", sender.msgs, wantKey)
				}
				if !sender.hadDeadline[0] {
					t.Fatal("send ctx has no deadline, want send timeout applied")
				}
				if len(repo.sentIDs) != 1 || repo.sentIDs[0] != outboxID1 {
					t.Fatalf("MarkSent ids = %v, want [%s]", repo.sentIDs, outboxID1)
				}
				wantLease := fixedNow.Add(defaultLeaseDuration)
				if !repo.leaseUntils[0].Equal(wantLease) {
					t.Fatalf("leaseUntil = %v, want %v", repo.leaseUntils[0], wantLease)
				}
			},
		},
		{
			name:   "retries on transient error using claimed attempt count",
			repo:   &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(outboxID1, 0)}},
			sender: &recordingSender{errs: []error{errors.New("temporary")}},
			cfg:    Config{BatchSize: 10},
			assert: func(t *testing.T, repo *fakeOutboxRepo, _ *recordingSender, stats ProcessStats) {
				t.Helper()
				if stats.Retried != 1 || stats.Failed != 0 {
					t.Fatalf("stats = %+v, want retried=1 failed=0", stats)
				}
				if len(repo.retryCalls) != 1 {
					t.Fatalf("retry calls = %d, want 1", len(repo.retryCalls))
				}
				wantNext := fixedNow.Add(1 * time.Minute)
				if got := repo.retryCalls[0].next; !got.Equal(wantNext) {
					t.Fatalf("next = %v, want %v", got, wantNext)
				}
				if repo.retryCalls[0].lastError != "temporary" {
					t.Fatalf("lastError = %q, want %q", repo.retryCalls[0].lastError, "temporary")
				}
			},
		},
		{
			name:   "marks failed on permanent error",
			repo:   &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(outboxID1, 0)}},
			sender: &recordingSender{errs: []error{domain.ErrMailPermanent}},
			cfg:    Config{BatchSize: 10},
			assert: func(t *testing.T, repo *fakeOutboxRepo, _ *recordingSender, stats ProcessStats) {
				t.Helper()
				if stats.Failed != 1 || stats.Retried != 0 {
					t.Fatalf("stats = %+v, want failed=1 retried=0", stats)
				}
				if len(repo.failedCalls) != 1 || repo.failedCalls[0].id != outboxID1 {
					t.Fatalf("failedCalls = %+v, want 1 for %s", repo.failedCalls, outboxID1)
				}
			},
		},
		{
			name: "marks failed when attempts are exhausted",
			// 5 回失敗済み → 今回の claim で 6 回目
			repo:   &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(outboxID1, 5)}},
			sender: &recordingSender{errs: []error{errors.New("still failing")}},
			cfg:    Config{BatchSize: 10},
			assert: func(t *testing.T, repo *fakeOutboxRepo, _ *recordingSender, stats ProcessStats) {
				t.Helper()
				if stats.Failed != 1 || len(repo.failedCalls) != 1 {
					t.Fatalf("stats = %+v failedCalls=%+v, want failed=1", stats, repo.failedCalls)
				}
				if len(repo.retryCalls) != 0 {
					t.Fatalf("retry calls = %+v, want none", repo.retryCalls)
				}
			},
		},
		{
			name:   "stops when nothing is pending",
			repo:   &fakeOutboxRepo{},
			sender: &recordingSender{},
			cfg:    Config{BatchSize: 10},
			assert: func(t *testing.T, repo *fakeOutboxRepo, _ *recordingSender, stats ProcessStats) {
				t.Helper()
				if stats.Processed != 0 || repo.claimCalls != 1 {
					t.Fatalf("stats = %+v claimCalls=%d, want processed=0 claimCalls=1", stats, repo.claimCalls)
				}
			},
		},
		{
			name: "halts batch and releases claim on auth error",
			repo: &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{
				sampleOutbox(outboxID1, 0), sampleOutbox(outboxID2, 0),
			}},
			sender: &recordingSender{errs: []error{domain.ErrMailAuth}},
			cfg:    Config{BatchSize: 10},
			assert: func(t *testing.T, repo *fakeOutboxRepo, sender *recordingSender, stats ProcessStats) {
				t.Helper()
				if stats.Halted != HaltAuthError || stats.Processed != 1 {
					t.Fatalf("stats = %+v, want halted=auth_error processed=1", stats)
				}
				if repo.claimCalls != 1 || len(sender.msgs) != 1 {
					t.Fatalf("claimCalls=%d sends=%d, want 1 and 1 (second row untouched)", repo.claimCalls, len(sender.msgs))
				}
				if len(repo.releaseCalls) != 1 || repo.releaseCalls[0].id != outboxID1 {
					t.Fatalf("releaseCalls = %+v, want 1 for %s", repo.releaseCalls, outboxID1)
				}
				wantNext := fixedNow.Add(authErrorRetryDelay)
				if !repo.releaseCalls[0].next.Equal(wantNext) {
					t.Fatalf("release next = %v, want %v", repo.releaseCalls[0].next, wantNext)
				}
				if len(repo.retryCalls) != 0 || len(repo.failedCalls) != 0 {
					t.Fatalf("retry=%+v failed=%+v, want neither (attempt must not be consumed)", repo.retryCalls, repo.failedCalls)
				}
			},
		},
		{
			name: "counts mark error and continues with next row",
			repo: &fakeOutboxRepo{
				pending:     []*domain.EmailOutboxMessage{sampleOutbox(outboxID1, 0), sampleOutbox(outboxID2, 0)},
				markSentErr: errors.New("db down"),
			},
			sender: &recordingSender{},
			cfg:    Config{BatchSize: 10},
			assert: func(t *testing.T, repo *fakeOutboxRepo, sender *recordingSender, stats ProcessStats) {
				t.Helper()
				if stats.Processed != 2 || stats.Errors != 2 || stats.Sent != 0 || stats.Halted != "" {
					t.Fatalf("stats = %+v, want processed=2 errors=2 sent=0", stats)
				}
				if len(sender.msgs) != 2 {
					t.Fatalf("sends = %d, want 2", len(sender.msgs))
				}
			},
		},
		{
			name:   "respects batch size",
			repo:   &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{sampleOutbox(outboxID1, 0), sampleOutbox(outboxID2, 0)}},
			sender: &recordingSender{},
			cfg:    Config{BatchSize: 1},
			assert: func(t *testing.T, repo *fakeOutboxRepo, _ *recordingSender, stats ProcessStats) {
				t.Helper()
				if stats.Processed != 1 || repo.claimCalls != 1 {
					t.Fatalf("stats = %+v claimCalls=%d, want processed=1 claimCalls=1", stats, repo.claimCalls)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDispatcher(tt.repo, tt.sender, tt.cfg)
			stats, err := d.ProcessPending(context.Background())
			if err != nil {
				t.Fatalf("ProcessPending() err = %v", err)
			}
			tt.assert(t, tt.repo, tt.sender, stats)
		})
	}
}

func TestDispatcherReturnsErrorWhenClaimFails(t *testing.T) {
	repo := &fakeOutboxRepo{claimErr: errors.New("connection refused")}
	d := newTestDispatcher(repo, &recordingSender{}, Config{BatchSize: 10})

	stats, err := d.ProcessPending(context.Background())
	if err == nil {
		t.Fatal("err = nil, want claim error")
	}
	if stats.Processed != 0 {
		t.Fatalf("processed = %d, want 0", stats.Processed)
	}
}

func TestDispatcherHaltsWhenTimeBudgetExceeded(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.EmailOutboxMessage{
		sampleOutbox(outboxID1, 0), sampleOutbox(outboxID2, 0),
	}}
	sender := &recordingSender{}
	d := NewDispatcher(repo, sender, Config{BatchSize: 10, TimeBudget: 100 * time.Second, SendTimeout: 30 * time.Second})

	// 1 通目の送信に 80 秒かかった想定。残り 20 秒 < 送信タイムアウト 30 秒なので 2 通目は claim しない
	clock := fixedNow
	d.now = func() time.Time { return clock }
	sender.onSend = func() { clock = clock.Add(80 * time.Second) }

	stats, err := d.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending() err = %v", err)
	}
	if stats.Halted != HaltTimeBudget || stats.Processed != 1 || stats.Sent != 1 {
		t.Fatalf("stats = %+v, want halted=time_budget processed=1 sent=1", stats)
	}
	if repo.claimCalls != 1 {
		t.Fatalf("claimCalls = %d, want 1", repo.claimCalls)
	}
}

func TestNewDispatcherKeepsLeaseLongerThanSendTimeout(t *testing.T) {
	d := NewDispatcher(&fakeOutboxRepo{}, &recordingSender{}, Config{SendTimeout: 10 * time.Minute, LeaseDuration: time.Minute})
	if d.leaseDuration <= d.sendTimeout {
		t.Fatalf("lease = %v, send timeout = %v; want lease > send timeout", d.leaseDuration, d.sendTimeout)
	}
}
