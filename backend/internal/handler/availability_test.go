package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

const testAvailabilityPath = "/api/v1/reservations/availability"

type handlerTestScheduleRepo struct {
	byDate map[string]domain.Schedule
}

func (r handlerTestScheduleRepo) Upsert(context.Context, domain.SetScheduleInput) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (r handlerTestScheduleRepo) FindByDate(_ context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	s, ok := r.byDate[date.String()]
	return s, ok, nil
}

func (r handlerTestScheduleRepo) ListStoredByYearMonth(context.Context, int, int) ([]domain.Schedule, error) {
	return nil, nil
}

type handlerTestReservationRepo struct {
	approvedByDate map[string]int
}

func (r handlerTestReservationRepo) Create(context.Context, domain.CreateReservationInput) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r handlerTestReservationRepo) GetAll(context.Context) ([]domain.Reservation, error) {
	return nil, nil
}

func (r handlerTestReservationRepo) List(context.Context, domain.ListReservationsFilter) ([]domain.Reservation, error) {
	return nil, nil
}

func (r handlerTestReservationRepo) GetByID(context.Context, uuid.UUID) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r handlerTestReservationRepo) UpdateStatus(context.Context, uuid.UUID, string) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r handlerTestReservationRepo) SumReservedPeopleByDate(_ context.Context, date datetime.Date) (int, error) {
	if r.approvedByDate == nil {
		return 0, nil
	}
	return r.approvedByDate[date.String()], nil
}

func (r handlerTestReservationRepo) SumReservedPeopleByDateRange(_ context.Context, from, to datetime.Date) (map[string]int, error) {
	if r.approvedByDate == nil {
		return map[string]int{}, nil
	}
	result := make(map[string]int)
	fromStr := from.String()
	toStr := to.String()
	for dateStr, count := range r.approvedByDate {
		if dateStr >= fromStr && dateStr <= toStr {
			result[dateStr] = count
		}
	}
	return result, nil
}

func newAvailabilityHandlerForTest(sched handlerTestScheduleRepo, res handlerTestReservationRepo) *AvailabilityHandler {
	resolver := service.NewScheduleResolver(sched)
	avail := service.NewAvailabilityService(resolver, res)
	getUC := usecase.NewGetAvailabilityUseCase(avail)
	return NewAvailabilityHandler(getUC, testAvailabilityPath)
}

func TestToAvailabilityResponse(t *testing.T) {
	t.Run("sets schedule_type to null on closed holiday", func(t *testing.T) {
		resp := toAvailabilityResponse(service.Availability{
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
		resp := toAvailabilityResponse(service.Availability{
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
		if !resp.IsHoliday {
			t.Fatal("IsHoliday = false, want true")
		}
	})

	t.Run("includes schedule_type pointer on bookable day", func(t *testing.T) {
		resp := toAvailabilityResponse(service.Availability{
			Date:         datetime.MustParseDate("2026-05-18"),
			ScheduleType: domain.ScheduleTypeNormal,
			Capacity:     10,
			Available:    4,
		})
		if resp.ScheduleType == nil {
			t.Fatal("ScheduleType = nil, want non-nil")
		}
		if *resp.ScheduleType != domain.ScheduleTypeNormal {
			t.Fatalf("ScheduleType = %q, want normal", *resp.ScheduleType)
		}
	})
}

func TestAvailabilityHandler_HandleAvailability(t *testing.T) {
	h := newAvailabilityHandlerForTest(handlerTestScheduleRepo{}, handlerTestReservationRepo{
		approvedByDate: map[string]int{"2026-05-18": 6},
	})

	t.Run("returns 200 with availability JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testAvailabilityPath+"?date=2026-05-18", nil)
		rec := httptest.NewRecorder()

		h.HandleAvailability(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		var body AvailabilityResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal() err = %v", err)
		}
		if body.Available != 4 {
			t.Fatalf("Available = %d, want 4", body.Available)
		}
		if body.ScheduleType == nil || *body.ScheduleType != domain.ScheduleTypeNormal {
			t.Fatalf("ScheduleType = %v, want normal pointer", body.ScheduleType)
		}
	})

	t.Run("returns 400 when date query is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testAvailabilityPath, nil)
		rec := httptest.NewRecorder()

		h.HandleAvailability(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		assertAvailabilityErrorCode(t, rec, ValidationErrorCode)
	})

	t.Run("returns 200 with monthly availability JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testAvailabilityPath+"?year=2026&month=5", nil)
		rec := httptest.NewRecorder()

		h.HandleAvailability(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		var body AvailabilityListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal() err = %v", err)
		}
		if body.Year != 2026 || body.Month != 5 {
			t.Fatalf("Year/Month = %d/%d, want 2026/5", body.Year, body.Month)
		}
		if len(body.Availabilities) != 31 {
			t.Fatalf("len(Availabilities) = %d, want 31", len(body.Availabilities))
		}
	})

	t.Run("returns 400 when month query is missing for monthly mode", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testAvailabilityPath+"?year=2026", nil)
		rec := httptest.NewRecorder()

		h.HandleAvailability(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("returns 400 when date and year/month are both specified", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testAvailabilityPath+"?date=2026-05-18&year=2026&month=5", nil)
		rec := httptest.NewRecorder()

		h.HandleAvailability(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		assertAvailabilityErrorCode(t, rec, ValidationErrorCode)
	})

	t.Run("returns 400 when date format is invalid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testAvailabilityPath+"?date=not-a-date", nil)
		rec := httptest.NewRecorder()

		h.HandleAvailability(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("returns 405 for non-GET method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, testAvailabilityPath+"?date=2026-05-18", nil)
		rec := httptest.NewRecorder()

		h.HandleAvailability(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})

	t.Run("returns holiday response with null schedule_type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testAvailabilityPath+"?date=2026-05-21", nil)
		rec := httptest.NewRecorder()

		h.HandleAvailability(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var body AvailabilityResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal() err = %v", err)
		}
		if !body.IsHoliday {
			t.Fatal("IsHoliday = false, want true")
		}
		if body.ScheduleType != nil {
			t.Fatalf("ScheduleType = %v, want nil", body.ScheduleType)
		}
		if body.Available != 0 || body.Capacity != 0 {
			t.Fatalf("capacity/available = %d/%d, want 0/0", body.Capacity, body.Available)
		}
	})
}

func assertAvailabilityErrorCode(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var payload ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() err = %v", err)
	}
	if payload.Error.Code != wantCode {
		t.Fatalf("error.code = %q, want %q", payload.Error.Code, wantCode)
	}
}
