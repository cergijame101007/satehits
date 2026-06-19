package mail

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

func sampleReservation() domain.Reservation {
	return domain.Reservation{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:      "山田太郎",
		People:    2,
		VisitDate: datetime.MustParseDate("2025-03-01"),
		VisitTime: datetime.MustParseTime("11:30"),
		Email:     "customer@example.com",
		Status:    "pending",
	}
}

func TestBuildReservationReceived(t *testing.T) {
	content := buildReservationReceived(sampleReservation())
	if content.Subject != "【さて、羊に戻るとしよう】ご予約を受け付けました" {
		t.Fatalf("Subject = %q", content.Subject)
	}
	if !strings.Contains(content.Text, "山田太郎 様") {
		t.Fatalf("Text = %q", content.Text)
	}
	if !strings.Contains(content.Text, noReplyFooter) {
		t.Fatalf("Text missing no-reply footer")
	}
}

func TestBuildReservationRejectedWithoutReason(t *testing.T) {
	content := buildReservationRejected(sampleReservation(), "")
	if strings.Contains(content.Text, "理由：") {
		t.Fatalf("Text should not contain reason label: %q", content.Text)
	}
}

func TestBuildReservationRejectedWithReason(t *testing.T) {
	content := buildReservationRejected(sampleReservation(), "定員超過のため")
	if !strings.Contains(content.Text, "理由：定員超過のため") {
		t.Fatalf("Text = %q", content.Text)
	}
}

func TestBuildReservationRejectedTrimsReason(t *testing.T) {
	content := buildReservationRejected(sampleReservation(), "  定員超過  ")
	if !strings.Contains(content.Text, "理由：定員超過") {
		t.Fatalf("Text = %q", content.Text)
	}
}

func TestBuildReservationApproved(t *testing.T) {
	content := buildReservationApproved(sampleReservation())
	if content.Subject != "【さて、羊に戻るとしよう】ご予約が確定しました" {
		t.Fatalf("Subject = %q", content.Subject)
	}
	if !strings.Contains(content.Text, cancelPolicyText) {
		t.Fatalf("Text missing cancel policy")
	}
}
