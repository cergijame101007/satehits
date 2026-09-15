package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
	inframail "github.com/cergijame101007/satehits/internal/infrastructure/mail"
)

// routingTestHandlers は 405 / 404 の判定だけを検証するためのハンドラ一式
// ルーティングは usecase を呼ぶ前に終わるため、fake の無い usecase は nil で渡す
type routingTestHandlers struct {
	reservation       http.HandlerFunc
	availability      http.HandlerFunc
	publicSchedules   http.HandlerFunc
	publicSuppliers   http.HandlerFunc
	adminSchedules    http.HandlerFunc
	adminReservations http.HandlerFunc
	adminSuppliers    http.HandlerFunc
	login             http.HandlerFunc
	refresh           http.HandlerFunc
	logout            http.HandlerFunc
	outbox            http.HandlerFunc
	root              http.HandlerFunc
}

func newRoutingTestHandlers(t *testing.T) routingTestHandlers {
	t.Helper()
	auth := newAuthHandlerWithDeps(t, authTestDeps{})
	dispatcher := inframail.NewDispatcher(emptyOutboxRepo{}, noopMailSender{}, inframail.Config{BatchSize: 5})
	return routingTestHandlers{
		reservation:     NewReservationHandler(nil, "/api/v1/reservations").HandleReservations,
		availability:    newAvailabilityHandlerForTest(handlerTestScheduleRepo{}, handlerTestReservationRepo{}).HandleAvailability,
		publicSchedules: newPublicScheduleHandlerForTest(handlerTestScheduleRepo{}, handlerTestReservationRepo{}).HandlePublicSchedules,
		publicSuppliers: NewPublicSupplierHandler(nil, "/api/v1/suppliers").HandlePublicSuppliers,
		adminSchedules: newScheduleHandlerForTest(&mutableHandlerTestScheduleRepo{
			byDate: map[string]domain.Schedule{},
		}).HandleSchedules,
		adminReservations: NewAdminReservationHandler(nil, nil, nil, "/api/v1/admin/reservations").HandleAdminReservations,
		adminSuppliers:    NewAdminSupplierHandler(nil, nil, nil, nil, nil, nil, nil, "/api/v1/admin/suppliers").HandleAdminSuppliers,
		login:             auth.HandleLogin,
		refresh:           auth.HandleRefresh,
		logout:            auth.HandleLogout,
		outbox:            NewOutboxHandler(dispatcher).HandleFlush,
		root:              HandleRoot,
	}
}

// docs/api_design.md §3: 受け付けないメソッドは 405 INVALID_REQUEST（Allow ヘッダ付き）の JSON
func TestHandlers_MethodNotAllowed(t *testing.T) {
	h := newRoutingTestHandlers(t)
	const reservationID = "00000000-0000-0000-0000-000000000001"

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		method    string
		path      string
		wantAllow string
	}{
		{name: "rejects GET on reservations", handler: h.reservation, method: http.MethodGet, path: "/api/v1/reservations", wantAllow: "POST"},
		{name: "rejects POST on availability", handler: h.availability, method: http.MethodPost, path: testAvailabilityPath, wantAllow: "GET"},
		{name: "rejects POST on public schedules", handler: h.publicSchedules, method: http.MethodPost, path: testPublicSchedulesPath, wantAllow: "GET"},
		{name: "rejects POST on public suppliers", handler: h.publicSuppliers, method: http.MethodPost, path: "/api/v1/suppliers", wantAllow: "GET"},
		{name: "rejects POST on admin schedules list", handler: h.adminSchedules, method: http.MethodPost, path: testAdminSchedulesPath, wantAllow: "GET"},
		{name: "rejects POST on admin schedule by date", handler: h.adminSchedules, method: http.MethodPost, path: testAdminSchedulesPath + "/2026-05-18", wantAllow: "GET, PUT, DELETE"},
		{name: "rejects PUT on admin reservations list", handler: h.adminReservations, method: http.MethodPut, path: "/api/v1/admin/reservations", wantAllow: "GET, POST"},
		{name: "rejects GET on admin reservation status", handler: h.adminReservations, method: http.MethodGet, path: "/api/v1/admin/reservations/" + reservationID + "/status", wantAllow: "PATCH"},
		{name: "rejects DELETE on admin suppliers list", handler: h.adminSuppliers, method: http.MethodDelete, path: "/api/v1/admin/suppliers", wantAllow: "GET, POST"},
		{name: "rejects GET on admin suppliers order", handler: h.adminSuppliers, method: http.MethodGet, path: "/api/v1/admin/suppliers/order", wantAllow: "PUT"},
		{name: "rejects POST on admin supplier by id", handler: h.adminSuppliers, method: http.MethodPost, path: "/api/v1/admin/suppliers/1", wantAllow: "GET, PUT, DELETE"},
		{name: "rejects GET on admin supplier image", handler: h.adminSuppliers, method: http.MethodGet, path: "/api/v1/admin/suppliers/1/image", wantAllow: "POST"},
		{name: "rejects GET on login", handler: h.login, method: http.MethodGet, path: testAuthLoginPath, wantAllow: "POST"},
		{name: "rejects GET on refresh", handler: h.refresh, method: http.MethodGet, path: testAuthRefreshPath, wantAllow: "POST"},
		{name: "rejects GET on logout", handler: h.logout, method: http.MethodGet, path: testAuthLogoutPath, wantAllow: "POST"},
		{name: "rejects GET on outbox flush", handler: h.outbox, method: http.MethodGet, path: "/internal/outbox/flush", wantAllow: "POST"},
		{name: "rejects POST on root", handler: h.root, method: http.MethodPost, path: "/", wantAllow: "GET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler(rec, httptest.NewRequest(tt.method, tt.path, nil))

			assertRoutingError(t, rec, http.StatusMethodNotAllowed, InvalidRequestCode, "許可されていないメソッドです")
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow = %q, want %q", got, tt.wantAllow)
			}
		})
	}
}

// docs/api_design.md §3: どのルートにも一致しないパスはメソッドにかかわらず 404 NOT_FOUND の JSON
func TestHandlers_UnknownPath(t *testing.T) {
	h := newRoutingTestHandlers(t)
	const reservationID = "00000000-0000-0000-0000-000000000001"

	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
	}{
		{name: "returns 404 for sub path of reservations", handler: h.reservation, method: http.MethodPost, path: "/api/v1/reservations/extra"},
		{name: "returns 404 for sub path of availability", handler: h.availability, method: http.MethodGet, path: testAvailabilityPath + "/extra"},
		{name: "returns 404 for sub path of public schedules", handler: h.publicSchedules, method: http.MethodGet, path: testPublicSchedulesPath + "/extra"},
		{name: "returns 404 for sub path of public suppliers", handler: h.publicSuppliers, method: http.MethodGet, path: "/api/v1/suppliers/extra"},
		{name: "returns 404 for admin schedules with trailing slash", handler: h.adminSchedules, method: http.MethodGet, path: testAdminSchedulesPath + "/"},
		{name: "returns 404 for nested path under admin schedule date", handler: h.adminSchedules, method: http.MethodGet, path: testAdminSchedulesPath + "/2026-05-18/extra"},
		{name: "returns 404 for admin schedules path without prefix match", handler: h.adminSchedules, method: http.MethodGet, path: testAdminSchedulesPath + "x"},
		{name: "returns 404 for admin reservations with trailing slash", handler: h.adminReservations, method: http.MethodGet, path: "/api/v1/admin/reservations/"},
		{name: "returns 404 for admin reservation by id", handler: h.adminReservations, method: http.MethodGet, path: "/api/v1/admin/reservations/" + reservationID},
		{name: "returns 404 for unknown action under admin reservation", handler: h.adminReservations, method: http.MethodPatch, path: "/api/v1/admin/reservations/" + reservationID + "/unknown"},
		{name: "returns 404 for admin suppliers with trailing slash", handler: h.adminSuppliers, method: http.MethodGet, path: "/api/v1/admin/suppliers/"},
		{name: "returns 404 for unknown action under admin supplier", handler: h.adminSuppliers, method: http.MethodPost, path: "/api/v1/admin/suppliers/1/unknown"},
		{name: "returns 404 for unknown path on root", handler: h.root, method: http.MethodGet, path: "/unknown"},
		{name: "returns 404 rather than 405 for unknown path with non-GET on root", handler: h.root, method: http.MethodPost, path: "/unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler(rec, httptest.NewRequest(tt.method, tt.path, nil))

			assertRoutingError(t, rec, http.StatusNotFound, NotFoundCode, "リソースが見つかりません")
			if got := rec.Header().Get("Allow"); got != "" {
				t.Fatalf("Allow = %q, want empty", got)
			}
		})
	}
}

func TestHandleRoot(t *testing.T) {
	t.Run("returns 200 JSON for GET /", func(t *testing.T) {
		rec := httptest.NewRecorder()
		HandleRoot(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); got != mediaTypeJSON {
			t.Fatalf("Content-Type = %q, want %q", got, mediaTypeJSON)
		}
		if got, want := rec.Body.String(), `{"message":"Welcome to the Go API","status":"success"}`; got != want {
			t.Fatalf("body = %s, want %s", got, want)
		}
	})
}

func assertRoutingError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode, wantMessage string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, wantStatus, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != mediaTypeJSON {
		t.Fatalf("Content-Type = %q, want %q", got, mediaTypeJSON)
	}
	var payload ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() err = %v; body = %s", err, rec.Body.String())
	}
	if payload.Error.Code != wantCode {
		t.Fatalf("error.code = %q, want %q", payload.Error.Code, wantCode)
	}
	if payload.Error.Message != wantMessage {
		t.Fatalf("error.message = %q, want %q", payload.Error.Message, wantMessage)
	}
}
