package usecase

import (
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
)

func validCreateAdminReservationCommand() CreateAdminReservationCommand {
	return CreateAdminReservationCommand{
		Name:      "山田太郎",
		People:    2,
		VisitDate: datetime.MustParseDate("2026-05-16"),
		VisitTime: datetime.MustParseTime("12:00"),
		Phone:     "090-1234-5678",
		Email:     "a@example.com",
		Source:    "phone",
	}
}

func TestValidateCreateAdminReservation(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*CreateAdminReservationCommand)
		wantField string
	}{
		{name: "accepts valid command with default status", mutate: func(cmd *CreateAdminReservationCommand) {}},
		{name: "rejects empty source", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Source = "" }, wantField: "source"},
		{name: "rejects invalid source", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Source = "invalid" }, wantField: "source"},
		{name: "rejects invalid status", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Status = "invalid" }, wantField: "status"},
		{name: "accepts explicit approved status", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Status = defaultAdminReservationStatus }},
		// 公開予約ポリシーを課さないことの回帰テスト（リードタイム・定休日・営業時間）
		{name: "accepts closed weekday (Thursday)", mutate: func(cmd *CreateAdminReservationCommand) {
			cmd.VisitDate = datetime.MustParseDate("2026-05-14")
		}},
		{name: "accepts date beyond 14-day window", mutate: func(cmd *CreateAdminReservationCommand) {
			cmd.VisitDate = datetime.MustParseDate("2026-12-31")
		}},
		{name: "accepts out-of-business-hours visit time", mutate: func(cmd *CreateAdminReservationCommand) {
			cmd.VisitTime = datetime.MustParseTime("23:00")
		}},
		// フォーマット/範囲検証は維持されること
		{name: "rejects empty name", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Name = "  " }, wantField: "name"},
		{name: "rejects people above maximum", mutate: func(cmd *CreateAdminReservationCommand) { cmd.People = 8 }, wantField: "people"},
		{name: "rejects zero visit_date", mutate: func(cmd *CreateAdminReservationCommand) { cmd.VisitDate = datetime.Date{} }, wantField: "visit_date"},
		{name: "rejects invalid email format", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Email = "not-an-email" }, wantField: "email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := validCreateAdminReservationCommand()
			tt.mutate(&cmd)
			violations := validateCreateAdminReservation(cmd)
			if tt.wantField == "" {
				assertNoViolations(t, violations)
				return
			}
			assertHasViolationField(t, violations, tt.wantField)
		})
	}
}

func TestValidateReservationStatusFilter(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		wantField string
	}{
		{name: "accepts empty status filter", status: ""},
		{name: "accepts pending status filter", status: "pending"},
		{name: "rejects invalid status filter", status: "invalid", wantField: "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validateReservationStatus(tt.status)
			if tt.wantField == "" {
				assertNoViolations(t, violations)
				return
			}
			assertHasViolationField(t, violations, tt.wantField)
		})
	}
}

func TestValidateReservationSourceOptional(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantField string
	}{
		{name: "accepts empty source filter", source: ""},
		{name: "accepts phone source filter", source: "phone"},
		{name: "rejects invalid source filter", source: "invalid", wantField: "source"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validateReservationSourceOptional(tt.source)
			if tt.wantField == "" {
				assertNoViolations(t, violations)
				return
			}
			assertHasViolationField(t, violations, tt.wantField)
		})
	}
}

func TestValidateUpdateStatusTarget(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		wantField string
	}{
		{name: "accepts approved target", status: "approved"},
		{name: "accepts no_show target", status: "no_show"},
		{name: "rejects empty status", status: "", wantField: "status"},
		{name: "rejects pending target", status: "pending", wantField: "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validateUpdateStatusTarget(tt.status)
			if tt.wantField == "" {
				assertNoViolations(t, violations)
				return
			}
			assertHasViolationField(t, violations, tt.wantField)
		})
	}
}

func TestValidateUpdateStatusReason(t *testing.T) {
	tests := []struct {
		name      string
		reason    string
		wantField string
	}{
		{name: "accepts empty reason", reason: ""},
		{name: "accepts reason at max length", reason: strings.Repeat("あ", maxRejectReasonRunes)},
		{name: "rejects reason over max length", reason: strings.Repeat("あ", maxRejectReasonRunes+1), wantField: "reason"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validateUpdateStatusReason(tt.reason)
			if tt.wantField == "" {
				assertNoViolations(t, violations)
				return
			}
			assertHasViolationField(t, violations, tt.wantField)
		})
	}
}
