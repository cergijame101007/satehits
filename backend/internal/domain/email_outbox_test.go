package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestIdempotencyKey(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	got := IdempotencyKey(MailTypeReservationApproved, id)
	want := "reservation_approved/550e8400-e29b-41d4-a716-446655440000"
	if got != want {
		t.Fatalf("IdempotencyKey() = %q, want %q", got, want)
	}
}
