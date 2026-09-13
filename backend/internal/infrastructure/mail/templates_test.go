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
		Phone:     "090-1234-5678",
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

func TestBuildPendingReminder(t *testing.T) {
	r := sampleReservation()
	r.Note = "窓側希望"
	content := buildPendingReminder(r, 3, "https://satehits.com/admin")

	if content.Subject != "【さて、羊に戻るとしよう】未承認のご予約があります（3/1 来店）" {
		t.Fatalf("Subject = %q", content.Subject)
	}
	for _, want := range []string{
		"来店日: 2025-03-01（土）",
		"来店時間: 11:30",
		"人数: 2 名",
		"お名前: 山田太郎",
		"電話番号: 090-1234-5678",
		"メール: customer@example.com",
		"備考: 窓側希望",
		"全部で 3 件",
		"管理画面: https://satehits.com/admin",
	} {
		if !strings.Contains(content.Text, want) {
			t.Fatalf("Text missing %q: %q", want, content.Text)
		}
	}
	if strings.Contains(content.Text, noReplyFooter) {
		t.Fatalf("owner mail should not carry the customer no-reply footer: %q", content.Text)
	}
	if !strings.Contains(content.HTML, "<p>") {
		t.Fatalf("HTML should be rendered: %q", content.HTML)
	}
}

func TestBuildPendingReminderShowsPlaceholderForEmptyNote(t *testing.T) {
	content := buildPendingReminder(sampleReservation(), 1, "http://localhost:4321/admin")
	if !strings.Contains(content.Text, "備考: なし") {
		t.Fatalf("Text = %q", content.Text)
	}
}
