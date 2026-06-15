package usecase

import (
	"strings"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// validCreateReservationCommand は公開予約 API の正常系ベース（2026-05-16 土・JST 基準）
func validCreateReservationCommand() CreateReservationCommand {
	return CreateReservationCommand{
		Name:      "山田太郎",
		People:    2,
		VisitDate: datetime.MustParseDate("2026-05-16"),
		VisitTime: datetime.MustParseTime("12:00"),
		Phone:     "090-1234-5678",
		Email:     "a@example.com",
	}
}

func assertNoViolations(t *testing.T, violations []FieldViolation) {
	t.Helper()
	if len(violations) != 0 {
		t.Fatalf("violations = %#v, want none", violations)
	}
}

func assertHasViolationField(t *testing.T, violations []FieldViolation, field string) {
	t.Helper()
	for _, v := range violations {
		if v.Field == field {
			return
		}
	}
	t.Fatalf("violations = %#v, want field %q", violations, field)
}

func TestValidEmail(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "accepts bare address", in: "a@example.com", want: true},
		{name: "accepts address with plus tag", in: "user.name+tag@example.co.jp", want: true},
		{name: "rejects display-name form", in: `田中 <tanaka@example.com>`, want: false},
		{name: "rejects stray angle bracket", in: "tanaka@example.com>", want: false},
		{name: "rejects over RFC 5321 max length", in: strings.Repeat("a", 250) + "@x.co", want: false},
		{name: "rejects empty string", in: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validEmail(tt.in)
			if got != tt.want {
				t.Fatalf("validEmail(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidPhone(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "accepts standard domestic format", in: "090-1234-5678", want: true},
		{name: "accepts fullwidth ideographic space", in: "090　1234　5678", want: true},
		{name: "accepts international format", in: "+81 (90) 1234-5678", want: true},
		{name: "rejects tab character", in: "090\t1234-5678", want: false},
		{name: "rejects newline", in: "090\n1234-5678", want: false},
		{name: "rejects too few runes", in: strings.Repeat("0", 9), want: false},
		{name: "accepts minimum digit count at min runes", in: strings.Repeat("0", 10), want: true},
		{name: "rejects over max runes", in: strings.Repeat("0-", 10) + strings.Repeat("-", 11), want: false},
		{name: "rejects empty string", in: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validPhone(tt.in)
			if got != tt.want {
				t.Fatalf("validPhone(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateCreateReservation(t *testing.T) {
	// 2026-05-15 12:00 JST → 翌日 5/16 から予約可、14 日先は 5/29 まで
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, storeLocation)

	t.Run("accepts valid command", func(t *testing.T) {
		assertNoViolations(t, validateCreateReservation(validCreateReservationCommand(), now))
	})

	t.Run("rejects whitespace-only name", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.Name = " \t　 "
		assertSingleViolationField(t, validateCreateReservation(cmd, now), "name")
	})

	t.Run("rejects people below minimum", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.People = 0
		assertHasViolationField(t, validateCreateReservation(cmd, now), "people")
	})

	t.Run("rejects people above maximum", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.People = 8
		assertHasViolationField(t, validateCreateReservation(cmd, now), "people")
	})

	t.Run("rejects zero visit_date", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.VisitDate = datetime.Date{}
		assertHasViolationField(t, validateCreateReservation(cmd, now), "visit_date")
	})

	t.Run("rejects zero visit_time", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.VisitTime = datetime.Time{}
		assertHasViolationField(t, validateCreateReservation(cmd, now), "visit_time")
	})

	t.Run("rejects empty phone", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.Phone = ""
		assertHasViolationField(t, validateCreateReservation(cmd, now), "phone")
	})

	t.Run("rejects empty email", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.Email = ""
		assertHasViolationField(t, validateCreateReservation(cmd, now), "email")
	})

	t.Run("rejects invalid email format", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.Email = "not-an-email"
		assertHasViolationField(t, validateCreateReservation(cmd, now), "email")
	})

	t.Run("rejects note over max rune length", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.Note = strings.Repeat("あ", maxNoteRunes+1)
		assertHasViolationField(t, validateCreateReservation(cmd, now), "note")
	})
}

func assertSingleViolationField(t *testing.T, violations []FieldViolation, field string) {
	t.Helper()
	if len(violations) != 1 {
		t.Fatalf("violations count = %d, want 1; violations = %#v", len(violations), violations)
	}
	if violations[0].Field != field {
		t.Fatalf("violations[0].Field = %q, want %q; violations = %#v", violations[0].Field, field, violations)
	}
}

func TestValidateCreateReservation_visitDateRange(t *testing.T) {
	// 範囲: 翌日〜14 日先（min=5/16, max=5/29）
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, storeLocation)
	cmd := validCreateReservationCommand()

	t.Run("accepts next bookable Saturday", func(t *testing.T) {
		cmd.VisitDate = datetime.MustParseDate("2026-05-16")
		assertNoViolations(t, validateCreateReservation(cmd, now))
	})

	t.Run("rejects same-day visit", func(t *testing.T) {
		cmd.VisitDate = datetime.MustParseDate("2026-05-15")
		assertHasViolationField(t, validateCreateReservation(cmd, now), "visit_date")
	})

	t.Run("rejects visit beyond 14-day window", func(t *testing.T) {
		cmd.VisitDate = datetime.MustParseDate("2026-05-30")
		assertHasViolationField(t, validateCreateReservation(cmd, now), "visit_date")
	})
}

func TestValidateCreateReservation_visitTimeWindow(t *testing.T) {
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, storeLocation) // 水
	monday := datetime.MustParseDate("2026-05-18")
	saturday := datetime.MustParseDate("2026-05-16")
	thursdayOverride := datetime.MustParseDate("2026-05-14")

	normalMonday := domain.Schedule{Date: monday, ScheduleType: domain.ScheduleTypeNormal}
	morningSaturday := domain.Schedule{Date: saturday, ScheduleType: domain.ScheduleTypeMorning}
	normalThursday := domain.Schedule{Date: thursdayOverride, ScheduleType: domain.ScheduleTypeNormal}

	t.Run("rejects weekday schedule before open time", func(t *testing.T) {
		v := validateVisitTimeForSchedule(normalMonday, datetime.MustParseTime("11:00"))
		assertHasViolationField(t, v, "visit_time")
	})

	t.Run("accepts Thursday override within weekday hours", func(t *testing.T) {
		v := validateVisitTimeForSchedule(normalThursday, datetime.MustParseTime("12:00"))
		assertNoViolations(t, v)
	})

	t.Run("rejects Thursday override before weekday open", func(t *testing.T) {
		v := validateVisitTimeForSchedule(normalThursday, datetime.MustParseTime("08:30"))
		assertHasViolationField(t, v, "visit_time")
	})

	t.Run("rejects Saturday morning schedule before 08:30", func(t *testing.T) {
		v := validateVisitTimeForSchedule(morningSaturday, datetime.MustParseTime("08:00"))
		assertHasViolationField(t, v, "visit_time")
	})

	t.Run("rejects weekday schedule after last order", func(t *testing.T) {
		v := validateVisitTimeForSchedule(normalMonday, datetime.MustParseTime("14:30"))
		assertHasViolationField(t, v, "visit_time")
	})

	t.Run("accepts weekday schedule at opening time", func(t *testing.T) {
		v := validateVisitTimeForSchedule(normalMonday, datetime.MustParseTime("11:30"))
		assertNoViolations(t, v)
	})

	t.Run("date range validation unchanged for bookable Saturday", func(t *testing.T) {
		cmd := validCreateReservationCommand()
		cmd.VisitDate = saturday
		cmd.VisitTime = datetime.MustParseTime("12:00")
		assertNoViolations(t, validateCreateReservation(cmd, now))
	})
}
