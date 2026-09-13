package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	reservationusecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

type stubPendingReminderReader struct {
	reservations []domain.Reservation
	err          error
}

func (s stubPendingReminderReader) ListPendingWebByVisitDateRange(context.Context, datetime.Date, datetime.Date) ([]domain.Reservation, error) {
	return s.reservations, s.err
}

func (s stubPendingReminderReader) CountPendingFrom(context.Context, datetime.Date) (int, error) {
	return len(s.reservations), nil
}

type stubPendingReminderEnqueuer struct {
	err error
}

func (s stubPendingReminderEnqueuer) EnqueuePendingReminder(context.Context, domain.Reservation, domain.MailType, int) error {
	return s.err
}

func newPendingReminderHandler(reader stubPendingReminderReader, enqueuer stubPendingReminderEnqueuer) *PendingReminderHandler {
	return NewPendingReminderHandler(reservationusecase.NewEnqueuePendingRemindersUseCase(reader, enqueuer))
}

func TestPendingReminderHandlerAcceptsPost(t *testing.T) {
	t.Run("returns counts as JSON", func(t *testing.T) {
		h := newPendingReminderHandler(stubPendingReminderReader{}, stubPendingReminderEnqueuer{})

		req := httptest.NewRequest(http.MethodPost, "/internal/reminders/pending", nil)
		rr := httptest.NewRecorder()
		h.HandlePendingReminders(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}
		var body map[string]int
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("json: %v", err)
		}
		for _, key := range []string{"candidates", "enqueued", "skipped"} {
			if got, ok := body[key]; !ok || got != 0 {
				t.Fatalf("body[%q] = %d (present=%v), want 0: %s", key, got, ok, rr.Body.String())
			}
		}
	})
}

func TestPendingReminderHandlerRejectsGet(t *testing.T) {
	h := newPendingReminderHandler(stubPendingReminderReader{}, stubPendingReminderEnqueuer{})

	req := httptest.NewRequest(http.MethodGet, "/internal/reminders/pending", nil)
	rr := httptest.NewRecorder()
	h.HandlePendingReminders(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
	var body ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Error.Code != InvalidRequestCode {
		t.Fatalf("code = %q, want %s", body.Error.Code, InvalidRequestCode)
	}
}

func TestPendingReminderHandlerReturns500OnFailure(t *testing.T) {
	h := newPendingReminderHandler(stubPendingReminderReader{err: errors.New("db down")}, stubPendingReminderEnqueuer{})

	req := httptest.NewRequest(http.MethodPost, "/internal/reminders/pending", nil)
	rr := httptest.NewRecorder()
	h.HandlePendingReminders(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	var body ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Error.Code != InternalErrorCode {
		t.Fatalf("code = %q, want %s", body.Error.Code, InternalErrorCode)
	}
}
