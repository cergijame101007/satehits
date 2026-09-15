package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

const testAdminReservationsPath = "/api/v1/admin/reservations"

var testAdminReservationUpdatedAt = time.Date(2026, 5, 1, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))

// adminReservationTestRepo は一覧・ステータス更新用の fake（ID で予約を引き、List の条件を記録する）
type adminReservationTestRepo struct {
	byID       map[uuid.UUID]domain.Reservation
	listed     []domain.Reservation
	listErr    error
	listCalls  int
	lastFilter domain.ListReservationsFilter
	updates    int
}

func (r *adminReservationTestRepo) Create(context.Context, domain.CreateReservationInput) (domain.Reservation, error) {
	return domain.Reservation{}, errors.New("not used")
}

func (r *adminReservationTestRepo) List(_ context.Context, f domain.ListReservationsFilter) ([]domain.Reservation, error) {
	r.listCalls++
	r.lastFilter = f
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listed, nil
}

func (r *adminReservationTestRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Reservation, error) {
	res, ok := r.byID[id]
	if !ok {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	return res, nil
}

func (r *adminReservationTestRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) (domain.Reservation, error) {
	res, ok := r.byID[id]
	if !ok {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	r.updates++
	res.Status = status
	res.UpdatedAt = testAdminReservationUpdatedAt
	r.byID[id] = res
	return res, nil
}

func (r *adminReservationTestRepo) SumReservedPeopleByDate(context.Context, datetime.Date) (int, error) {
	return 0, nil
}

func (r *adminReservationTestRepo) SumReservedPeopleByDateRange(context.Context, datetime.Date, datetime.Date) (map[string]int, error) {
	return map[string]int{}, nil
}

var _ domain.ReservationRepository = (*adminReservationTestRepo)(nil)

func testAdminReservation(id string, status string) domain.Reservation {
	return domain.Reservation{
		ID:        uuid.MustParse(id),
		Name:      "山田太郎",
		People:    2,
		VisitDate: datetime.MustParseDate("2026-05-20"),
		VisitTime: datetime.MustParseTime("12:00"),
		Phone:     "090-1234-5678",
		Email:     "yamada@example.com",
		Status:    status,
		Source:    "web",
		CreatedAt: testAdminReservationUpdatedAt,
		UpdatedAt: testAdminReservationUpdatedAt,
	}
}

// newAdminReservationHandlerForTest は一覧・ステータス更新だけを使う（手動登録の UseCase は nil）
func newAdminReservationHandlerForTest(repo *adminReservationTestRepo) *AdminReservationHandler {
	return NewAdminReservationHandler(
		usecase.NewListReservationsUseCase(repo),
		nil,
		usecase.NewUpdateReservationStatusUseCase(repo, &authTestTxManager{}, nil),
		testAdminReservationsPath,
	)
}

func newUpdateStatusRequest(id, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPatch, testAdminReservationsPath+"/"+id+"/status", strings.NewReader(body))
	req.Header.Set("Content-Type", mediaTypeJSON)
	return req
}

// assertAdminReservationError はステータス・error.code・details の field 一覧を確認する
func assertAdminReservationError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string, wantFields ...string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, wantStatus, rec.Body.String())
	}
	var payload ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() err = %v; body = %s", err, rec.Body.String())
	}
	if payload.Error.Code != wantCode {
		t.Fatalf("error.code = %q, want %q", payload.Error.Code, wantCode)
	}
	if len(payload.Error.Details) != len(wantFields) {
		t.Fatalf("error.details = %+v, want fields %v", payload.Error.Details, wantFields)
	}
	for i, field := range wantFields {
		if payload.Error.Details[i].Field != field {
			t.Fatalf("error.details[%d].field = %q, want %q", i, payload.Error.Details[i].Field, field)
		}
	}
}

func TestAdminReservationHandler_UpdateStatus(t *testing.T) {
	const (
		pendingID  = "550e8400-e29b-41d4-a716-446655440001"
		approvedID = "550e8400-e29b-41d4-a716-446655440002"
		missingID  = "550e8400-e29b-41d4-a716-446655440099"
	)
	newRepo := func() *adminReservationTestRepo {
		return &adminReservationTestRepo{byID: map[uuid.UUID]domain.Reservation{
			uuid.MustParse(pendingID):  testAdminReservation(pendingID, "pending"),
			uuid.MustParse(approvedID): testAdminReservation(approvedID, "approved"),
		}}
	}

	t.Run("returns 200 with updated status", func(t *testing.T) {
		repo := newRepo()
		h := newAdminReservationHandlerForTest(repo)

		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, newUpdateStatusRequest(pendingID, `{"status":"approved"}`))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		payload := decodeJSONObject(t, rec.Body.Bytes())
		assertJSONKeys(t, payload, "id", "status", "updated_at")
		var body UpdateStatusResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal() err = %v", err)
		}
		if body.ID.String() != pendingID || body.Status != "approved" {
			t.Fatalf("body = %+v, want id=%s status=approved", body, pendingID)
		}
		if body.UpdatedAt != "2026-05-01T12:00:00+09:00" {
			t.Fatalf("updated_at = %q, want 2026-05-01T12:00:00+09:00", body.UpdatedAt)
		}
	})

	t.Run("returns 404 NOT_FOUND when reservation does not exist", func(t *testing.T) {
		repo := newRepo()
		h := newAdminReservationHandlerForTest(repo)

		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, newUpdateStatusRequest(missingID, `{"status":"approved"}`))

		assertAdminReservationError(t, rec, http.StatusNotFound, NotFoundCode)
		if repo.updates != 0 {
			t.Fatalf("UpdateStatus calls = %d, want 0", repo.updates)
		}
	})

	t.Run("returns 400 INVALID_REQUEST for malformed UUID", func(t *testing.T) {
		repo := newRepo()
		h := newAdminReservationHandlerForTest(repo)

		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, newUpdateStatusRequest("not-a-uuid", `{"status":"approved"}`))

		assertAdminReservationError(t, rec, http.StatusBadRequest, InvalidRequestCode)
	})

	t.Run("returns 400 INVALID_REQUEST for non-JSON content type", func(t *testing.T) {
		repo := newRepo()
		h := newAdminReservationHandlerForTest(repo)

		req := newUpdateStatusRequest(pendingID, `{"status":"approved"}`)
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, req)

		assertAdminReservationError(t, rec, http.StatusBadRequest, InvalidRequestCode)
	})

	t.Run("returns 400 VALIDATION_ERROR for disallowed transition", func(t *testing.T) {
		repo := newRepo()
		h := newAdminReservationHandlerForTest(repo)

		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, newUpdateStatusRequest(approvedID, `{"status":"rejected"}`))

		assertAdminReservationError(t, rec, http.StatusBadRequest, ValidationErrorCode, "status")
		if repo.updates != 0 {
			t.Fatalf("UpdateStatus calls = %d, want 0", repo.updates)
		}
	})

	t.Run("returns 405 for GET on status path", func(t *testing.T) {
		h := newAdminReservationHandlerForTest(newRepo())

		req := httptest.NewRequest(http.MethodGet, testAdminReservationsPath+"/"+pendingID+"/status", nil)
		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})
}

func TestAdminReservationHandler_List(t *testing.T) {
	t.Run("returns 200 with reservations and total", func(t *testing.T) {
		repo := &adminReservationTestRepo{listed: []domain.Reservation{
			testAdminReservation("550e8400-e29b-41d4-a716-446655440001", "pending"),
			testAdminReservation("550e8400-e29b-41d4-a716-446655440002", "approved"),
		}}
		h := newAdminReservationHandlerForTest(repo)

		req := httptest.NewRequest(http.MethodGet, testAdminReservationsPath+"?date=2026-05-20&status=pending&source=web", nil)
		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		assertJSONKeys(t, decodeJSONObject(t, rec.Body.Bytes()), "reservations", "total")
		var body ReservationListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal() err = %v", err)
		}
		if body.Total != 2 || len(body.Reservations) != 2 {
			t.Fatalf("total = %d, len(reservations) = %d, want 2 and 2", body.Total, len(body.Reservations))
		}
		if body.Reservations[0].Status != "pending" || body.Reservations[1].Status != "approved" {
			t.Fatalf("reservations = %+v, want repository order", body.Reservations)
		}
		f := repo.lastFilter
		if f.Date == nil || f.Date.String() != "2026-05-20" || f.Status != "pending" || f.Source != "web" {
			t.Fatalf("filter = {Date:%v Status:%q Source:%q}, want 2026-05-20/pending/web", f.Date, f.Status, f.Source)
		}
	})

	t.Run("returns empty array when no reservations", func(t *testing.T) {
		h := newAdminReservationHandlerForTest(&adminReservationTestRepo{})

		req := httptest.NewRequest(http.MethodGet, testAdminReservationsPath+"?date=2026-05-20", nil)
		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		if got := strings.TrimSpace(rec.Body.String()); got != `{"reservations":[],"total":0}` {
			t.Fatalf("body = %s, want {\"reservations\":[],\"total\":0}", got)
		}
	})

	t.Run("returns 400 VALIDATION_ERROR for malformed date", func(t *testing.T) {
		repo := &adminReservationTestRepo{}
		h := newAdminReservationHandlerForTest(repo)

		req := httptest.NewRequest(http.MethodGet, testAdminReservationsPath+"?date=2026/05/20", nil)
		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, req)

		assertAdminReservationError(t, rec, http.StatusBadRequest, ValidationErrorCode, "date")
		if repo.listCalls != 0 {
			t.Fatalf("List calls = %d, want 0", repo.listCalls)
		}
	})

	t.Run("returns 400 VALIDATION_ERROR for unknown status", func(t *testing.T) {
		repo := &adminReservationTestRepo{}
		h := newAdminReservationHandlerForTest(repo)

		req := httptest.NewRequest(http.MethodGet, testAdminReservationsPath+"?status=done", nil)
		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, req)

		assertAdminReservationError(t, rec, http.StatusBadRequest, ValidationErrorCode, "status")
		if repo.listCalls != 0 {
			t.Fatalf("List calls = %d, want 0", repo.listCalls)
		}
	})

	t.Run("returns 500 INTERNAL_ERROR when repository fails", func(t *testing.T) {
		captureLog(t)
		h := newAdminReservationHandlerForTest(&adminReservationTestRepo{listErr: errors.New("db down")})

		req := httptest.NewRequest(http.MethodGet, testAdminReservationsPath, nil)
		rec := httptest.NewRecorder()
		h.HandleAdminReservations(rec, req)

		assertAdminReservationError(t, rec, http.StatusInternalServerError, InternalErrorCode)
	})
}
