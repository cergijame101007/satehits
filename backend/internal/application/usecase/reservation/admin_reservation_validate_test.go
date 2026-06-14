package usecase

import (
	"testing"
	"time"

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
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, storeLocation)

	tests := []struct {
		name      string
		mutate    func(*CreateAdminReservationCommand)
		wantField string
	}{
		{name: "accepts valid command with default status", mutate: func(cmd *CreateAdminReservationCommand) {}},
		{name: "rejects empty source", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Source = "" }, wantField: "source"},
		{name: "rejects invalid source", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Source = "invalid" }, wantField: "source"},
		{name: "rejects invalid status", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Status = "invalid" }, wantField: "status"},
		{name: "accepts explicit approved status", mutate: func(cmd *CreateAdminReservationCommand) { cmd.Status = "approved" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := validCreateAdminReservationCommand()
			tt.mutate(&cmd)
			violations := validateCreateAdminReservation(cmd, now)
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
