package mail

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
)

type fakeOutboxRepoForEnqueuer struct {
	last domain.EnqueueEmailInput
	err  error
}

func (f *fakeOutboxRepoForEnqueuer) Enqueue(_ context.Context, in domain.EnqueueEmailInput) error {
	f.last = in
	return f.err
}

func (f *fakeOutboxRepoForEnqueuer) ClaimNextPending(context.Context, time.Time) (*domain.EmailOutboxMessage, error) {
	return nil, nil
}

func (f *fakeOutboxRepoForEnqueuer) MarkSent(context.Context, uuid.UUID) error { return nil }

func (f *fakeOutboxRepoForEnqueuer) MarkRetry(context.Context, uuid.UUID, time.Time, string) error {
	return nil
}

func (f *fakeOutboxRepoForEnqueuer) MarkFailed(context.Context, uuid.UUID, string) error {
	return nil
}

func (f *fakeOutboxRepoForEnqueuer) ReleaseClaim(context.Context, uuid.UUID, time.Time, string) error {
	return nil
}

func TestOutboxEnqueuerReservationReceived(t *testing.T) {
	repo := &fakeOutboxRepoForEnqueuer{}
	e := NewOutboxEnqueuer(repo, EnqueuerConfig{FromAddress: "さて、羊に戻るとしよう <noreply@satehits.com>"})
	r := sampleReservation()

	if err := e.EnqueueReservationReceived(context.Background(), r); err != nil {
		t.Fatalf("EnqueueReservationReceived() err = %v", err)
	}
	if repo.last.MailType != domain.MailTypeReservationReceived {
		t.Fatalf("MailType = %q", repo.last.MailType)
	}
	if repo.last.ReservationID != r.ID {
		t.Fatalf("ReservationID = %v", repo.last.ReservationID)
	}
	if repo.last.ToAddress != r.Email {
		t.Fatalf("ToAddress = %q", repo.last.ToAddress)
	}
	if !strings.Contains(repo.last.Subject, "受け付けました") {
		t.Fatalf("Subject = %q", repo.last.Subject)
	}
	if repo.last.BodyText == "" || repo.last.BodyHTML == "" {
		t.Fatal("body should be rendered")
	}
}

func TestOutboxEnqueuerReservationRejectedIncludesReason(t *testing.T) {
	repo := &fakeOutboxRepoForEnqueuer{}
	e := NewOutboxEnqueuer(repo, EnqueuerConfig{FromAddress: "noreply@satehits.com"})

	if err := e.EnqueueReservationRejected(context.Background(), sampleReservation(), "定員超過"); err != nil {
		t.Fatalf("err = %v", err)
	}
	if repo.last.MailType != domain.MailTypeReservationRejected {
		t.Fatalf("MailType = %q", repo.last.MailType)
	}
	if !strings.Contains(repo.last.BodyText, "定員超過") {
		t.Fatalf("BodyText missing reason: %q", repo.last.BodyText)
	}
}

func TestOutboxEnqueuerPropagatesAlreadyEnqueued(t *testing.T) {
	repo := &fakeOutboxRepoForEnqueuer{err: domain.ErrMailAlreadyEnqueued}
	e := NewOutboxEnqueuer(repo, EnqueuerConfig{FromAddress: "noreply@satehits.com"})

	err := e.EnqueueReservationApproved(context.Background(), sampleReservation())
	if !errors.Is(err, domain.ErrMailAlreadyEnqueued) {
		t.Fatalf("err = %v, want ErrMailAlreadyEnqueued", err)
	}
}

func TestOutboxEnqueuerPendingReminder(t *testing.T) {
	repo := &fakeOutboxRepoForEnqueuer{}
	e := NewOutboxEnqueuer(repo, EnqueuerConfig{
		FromAddress:  "noreply@satehits.com",
		OwnerAddress: "owner@example.com",
		AdminURL:     "https://satehits.com/admin",
	})
	r := sampleReservation()

	if err := e.EnqueuePendingReminder(context.Background(), r, domain.MailTypePendingReminder3D, 4); err != nil {
		t.Fatalf("EnqueuePendingReminder() err = %v", err)
	}
	if repo.last.MailType != domain.MailTypePendingReminder3D {
		t.Fatalf("MailType = %q, want pending_reminder_3d", repo.last.MailType)
	}
	if repo.last.ReservationID != r.ID {
		t.Fatalf("ReservationID = %v", repo.last.ReservationID)
	}
	if repo.last.ToAddress != "owner@example.com" {
		t.Fatalf("ToAddress = %q, want owner address (not the customer)", repo.last.ToAddress)
	}
	if repo.last.FromAddress != "noreply@satehits.com" {
		t.Fatalf("FromAddress = %q", repo.last.FromAddress)
	}
	if !strings.Contains(repo.last.BodyText, "全部で 4 件") || !strings.Contains(repo.last.BodyText, "https://satehits.com/admin") {
		t.Fatalf("BodyText = %q", repo.last.BodyText)
	}
}

func TestOutboxEnqueuerPendingReminderRequiresOwnerAddress(t *testing.T) {
	repo := &fakeOutboxRepoForEnqueuer{}
	e := NewOutboxEnqueuer(repo, EnqueuerConfig{FromAddress: "noreply@satehits.com"})

	err := e.EnqueuePendingReminder(context.Background(), sampleReservation(), domain.MailTypePendingReminder1D, 1)
	if !errors.Is(err, ErrOwnerAddressNotConfigured) {
		t.Fatalf("err = %v, want ErrOwnerAddressNotConfigured", err)
	}
	if repo.last.MailType != "" {
		t.Fatal("repo.Enqueue should not be called without owner address")
	}
}

func TestOutboxEnqueuerPendingReminderPropagatesAlreadyEnqueued(t *testing.T) {
	repo := &fakeOutboxRepoForEnqueuer{err: domain.ErrMailAlreadyEnqueued}
	e := NewOutboxEnqueuer(repo, EnqueuerConfig{FromAddress: "noreply@satehits.com", OwnerAddress: "owner@example.com"})

	err := e.EnqueuePendingReminder(context.Background(), sampleReservation(), domain.MailTypePendingReminder1D, 1)
	if !errors.Is(err, domain.ErrMailAlreadyEnqueued) {
		t.Fatalf("err = %v, want ErrMailAlreadyEnqueued", err)
	}
}
