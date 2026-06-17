package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

const testPublicSchedulesPath = "/api/v1/schedules"

func newPublicScheduleHandlerForTest(sched handlerTestScheduleRepo, res handlerTestReservationRepo) *PublicScheduleHandler {
	resolver := service.NewScheduleResolver(sched)
	avail := service.NewAvailabilityService(resolver, res)
	getUC := usecase.NewGetAvailabilityUseCase(avail)
	return NewPublicScheduleHandler(getUC, testPublicSchedulesPath)
}

func TestToDaySchedule(t *testing.T) {
	t.Run("sets schedule_type to null on closed holiday", func(t *testing.T) {
		resp := toDaySchedule(service.Availability{
			Date:         datetime.MustParseDate("2026-05-21"),
			ScheduleType: domain.ScheduleTypeClosed,
			IsHoliday:    true,
		})
		if resp.ScheduleType != nil {
			t.Fatalf("ScheduleType = %v, want nil", resp.ScheduleType)
		}
		if !resp.IsHoliday {
			t.Fatal("IsHoliday = false, want true")
		}
	})

	t.Run("preserves schedule_type and event fields on external_event holiday", func(t *testing.T) {
		resp := toDaySchedule(service.Availability{
			Date:             datetime.MustParseDate("2026-02-11"),
			ScheduleType:     domain.ScheduleTypeExternalEvent,
			EventName:        "和紅茶をしばく会",
			EventDescription: "入門編@WINE LAB",
			IsHoliday:        true,
			Capacity:         0,
			Available:        0,
		})
		if resp.ScheduleType == nil || *resp.ScheduleType != domain.ScheduleTypeExternalEvent {
			t.Fatalf("ScheduleType = %v, want external_event pointer", resp.ScheduleType)
		}
		if resp.EventName != "和紅茶をしばく会" {
			t.Fatalf("EventName = %q", resp.EventName)
		}
		if resp.EventDescription != "入門編@WINE LAB" {
			t.Fatalf("EventDescription = %q", resp.EventDescription)
		}
		if !resp.IsHoliday {
			t.Fatal("IsHoliday = false, want true")
		}
	})

	t.Run("includes schedule_type on bookable day", func(t *testing.T) {
		resp := toDaySchedule(service.Availability{
			ScheduleType: domain.ScheduleTypeNormal,
			Capacity:     10,
			Available:    4,
		})
		if resp.ScheduleType == nil || *resp.ScheduleType != domain.ScheduleTypeNormal {
			t.Fatalf("ScheduleType = %v, want normal pointer", resp.ScheduleType)
		}
		if resp.Available != 4 {
			t.Fatalf("Available = %d, want 4", resp.Available)
		}
	})
}

func TestPublicScheduleHandler_HandlePublicSchedules(t *testing.T) {
	h := newPublicScheduleHandlerForTest(handlerTestScheduleRepo{}, handlerTestReservationRepo{
		approvedByDate: map[string]int{"2026-05-18": 6},
	})

	t.Run("returns 200 with monthly schedule JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testPublicSchedulesPath+"?year=2026&month=5", nil)
		rec := httptest.NewRecorder()

		h.HandlePublicSchedules(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		var body MonthlyScheduleResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal() err = %v", err)
		}
		if body.Year != 2026 || body.Month != 5 {
			t.Fatalf("Year/Month = %d/%d, want 2026/5", body.Year, body.Month)
		}
		if len(body.Schedules) != 31 {
			t.Fatalf("len(Schedules) = %d, want 31", len(body.Schedules))
		}
	})

	t.Run("returns 400 when year query is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testPublicSchedulesPath+"?month=5", nil)
		rec := httptest.NewRecorder()

		h.HandlePublicSchedules(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		assertAvailabilityErrorCode(t, rec, ValidationErrorCode)
	})

	t.Run("returns 400 when month query is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testPublicSchedulesPath+"?year=2026", nil)
		rec := httptest.NewRecorder()

		h.HandlePublicSchedules(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("returns 405 for non-GET method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, testPublicSchedulesPath+"?year=2026&month=5", nil)
		rec := httptest.NewRecorder()

		h.HandlePublicSchedules(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})
}
