package usecase

import (
	"strings"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

func TestValidEmail(t *testing.T) {
	if !validEmail("a@example.com") {
		t.Fatal("expected bare address")
	}
	if validEmail(`田中 <tanaka@example.com>`) {
		t.Fatal("expected display-name form rejected")
	}
	if validEmail("tanaka@example.com>") {
		t.Fatal("expected stray bracket rejected")
	}
	if !validEmail("user.name+tag@example.co.jp") {
		t.Fatal("expected bare address with plus tag")
	}
	longLocal := strings.Repeat("a", 250)
	if validEmail(longLocal + "@x.co") {
		t.Fatal("expected over RFC 5321 length rejected")
	}
}

func TestValidPhone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"standard", "090-1234-5678", true},
		{"fullwidth_space", "090　1234　5678", true},
		{"intl_format", "+81 (90) 1234-5678", true},
		{"tab", "090\t1234-5678", false},
		{"newline", "090\n1234-5678", false},
		{"too_few_runes", strings.Repeat("0", 9), false},
		{"min_runes_digits", strings.Repeat("0", 10), true},
		{"over_max_runes", strings.Repeat("0-", 10) + strings.Repeat("-", 11), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := validPhone(tt.in); got != tt.want {
				t.Errorf("validPhone(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateCreateReservation_visitDateRange(t *testing.T) {
	// 2026-05-15 は金曜（JST）→ 来店日としては定休扱いだが、範囲外チェックは min=5/16 max=5/29
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, storeLocation)

	cmd := CreateReservationCommand{
		Name:      "山田",
		People:    2,
		VisitDate: datetime.MustParseDate("2026-05-16"), // 土・翌日
		VisitTime: datetime.MustParseTime("12:00"),
		Phone:     "090-1234-5678",
		Email:     "a@example.com",
	}
	if v := validateCreateReservation(cmd, now); len(v) != 0 {
		t.Fatalf("expected no violations, got %#v", v)
	}

	cmd.VisitDate = datetime.MustParseDate("2026-05-15") // 当日不可
	if v := validateCreateReservation(cmd, now); len(v) == 0 {
		t.Fatal("expected visit_date violation for same-day")
	}

	cmd.VisitDate = datetime.MustParseDate("2026-05-30") // 15日翌日から14日先は 5/29 まで
	if v := validateCreateReservation(cmd, now); len(v) == 0 {
		t.Fatal("expected visit_date violation for too far")
	}
}

func TestValidateCreateReservation_thursdayClosed(t *testing.T) {
	// 2026-05-13 は水曜 → 翌日木は定休
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, storeLocation)
	cmd := CreateReservationCommand{
		Name:      "山田",
		People:    2,
		VisitDate: datetime.MustParseDate("2026-05-14"), // 木
		VisitTime: datetime.MustParseTime("12:00"),
		Phone:     "090-1234-5678",
		Email:     "a@example.com",
	}
	v := validateCreateReservation(cmd, now)
	if len(v) == 0 {
		t.Fatal("expected visit_date violation for Thursday")
	}
}

func TestValidateCreateReservation_visitTimeWindow(t *testing.T) {
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, storeLocation) // 水
	cmd := CreateReservationCommand{
		Name:      "山田",
		People:    2,
		VisitDate: datetime.MustParseDate("2026-05-14"), // 木は定休のため visit_time は検証しない（visit_date で弾く）
		VisitTime: datetime.MustParseTime("08:30"),
		Phone:     "090-1234-5678",
		Email:     "a@example.com",
	}
	if v := validateCreateReservation(cmd, now); len(v) == 0 {
		t.Fatal("expected at least visit_date violation")
	}

	cmd.VisitDate = datetime.MustParseDate("2026-05-16") // 土
	cmd.VisitTime = datetime.MustParseTime("08:00")
	if v := validateCreateReservation(cmd, now); len(v) == 0 {
		t.Fatal("expected visit_time violation before 8:30 on Saturday")
	}
}
