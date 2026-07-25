package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/schedule"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

const testAdminSchedulesPath = "/api/v1/admin/schedules"

// mutableHandlerTestScheduleRepo は DELETE 統合テスト用（行の追加・削除が可能）
type mutableHandlerTestScheduleRepo struct {
	byDate map[string]domain.Schedule
}

func (r *mutableHandlerTestScheduleRepo) Upsert(_ context.Context, in domain.SetScheduleInput) (domain.Schedule, bool, error) {
	out := domain.Schedule{
		Date:             in.Date,
		ScheduleType:     in.ScheduleType,
		Capacity:         in.Capacity,
		EventName:        in.EventName,
		EventDescription: in.EventDescription,
		OpenTime:         in.OpenTime,
		LastOrderTime:    in.LastOrderTime,
		CloseTime:        in.CloseTime,
	}
	_, existed := r.byDate[in.Date.String()]
	if r.byDate == nil {
		r.byDate = map[string]domain.Schedule{}
	}
	r.byDate[in.Date.String()] = out
	return out, !existed, nil
}

func (r *mutableHandlerTestScheduleRepo) FindByDate(_ context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	s, ok := r.byDate[date.String()]
	return s, ok, nil
}

func (r *mutableHandlerTestScheduleRepo) ListStoredByYearMonth(context.Context, int, int) ([]domain.Schedule, error) {
	return nil, nil
}

func (r *mutableHandlerTestScheduleRepo) DeleteByDate(_ context.Context, date datetime.Date) error {
	if _, ok := r.byDate[date.String()]; !ok {
		return domain.ErrScheduleNotStored
	}
	delete(r.byDate, date.String())
	return nil
}

func newScheduleHandlerForTest(repo *mutableHandlerTestScheduleRepo) *ScheduleHandler {
	resolver := service.NewScheduleResolver(repo)
	return NewScheduleHandler(
		usecase.NewSetScheduleUseCase(repo),
		usecase.NewListSchedulesUseCase(resolver),
		usecase.NewGetScheduleUseCase(resolver),
		usecase.NewDeleteScheduleUseCase(repo),
		testAdminSchedulesPath,
	)
}

func TestScheduleHandler_HandleSchedules_Delete(t *testing.T) {
	t.Run("returns 204 when stored row is deleted", func(t *testing.T) {
		d := datetime.MustParseDate("2026-05-18")
		repo := &mutableHandlerTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				d.String(): {Date: d, ScheduleType: domain.ScheduleTypeEvent, Capacity: 10},
			},
		}
		h := newScheduleHandlerForTest(repo)

		req := httptest.NewRequest(http.MethodDelete, testAdminSchedulesPath+"/2026-05-18", nil)
		rec := httptest.NewRecorder()
		h.HandleSchedules(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body = %s", rec.Code, rec.Body.String())
		}
		if _, ok := repo.byDate[d.String()]; ok {
			t.Fatal("row still present after delete")
		}
	})

	t.Run("returns 404 when row is not stored", func(t *testing.T) {
		h := newScheduleHandlerForTest(&mutableHandlerTestScheduleRepo{
			byDate: map[string]domain.Schedule{},
		})

		req := httptest.NewRequest(http.MethodDelete, testAdminSchedulesPath+"/2026-05-18", nil)
		rec := httptest.NewRecorder()
		h.HandleSchedules(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
		}
		assertScheduleErrorCode(t, rec, NotFoundCode)
	})

	t.Run("returns 405 when DELETE targets list path", func(t *testing.T) {
		h := newScheduleHandlerForTest(&mutableHandlerTestScheduleRepo{byDate: map[string]domain.Schedule{}})

		req := httptest.NewRequest(http.MethodDelete, testAdminSchedulesPath, nil)
		rec := httptest.NewRecorder()
		h.HandleSchedules(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})

	t.Run("returns 400 for invalid date path", func(t *testing.T) {
		h := newScheduleHandlerForTest(&mutableHandlerTestScheduleRepo{byDate: map[string]domain.Schedule{}})

		req := httptest.NewRequest(http.MethodDelete, testAdminSchedulesPath+"/not-a-date", nil)
		rec := httptest.NewRecorder()
		h.HandleSchedules(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
		}
		assertScheduleErrorCode(t, rec, InvalidRequestCode)
	})
}

func assertScheduleErrorCode(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var payload ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() err = %v", err)
	}
	if payload.Error.Code != wantCode {
		t.Fatalf("error.code = %q, want %q", payload.Error.Code, wantCode)
	}
}

func TestResolveSetScheduleCapacity(t *testing.T) {
	t.Run("omitted uses default 10", func(t *testing.T) {
		if got := resolveSetScheduleCapacity(nil); got != defaultSetScheduleCapacity {
			t.Fatalf("got %d, want %d", got, defaultSetScheduleCapacity)
		}
	})

	t.Run("explicit zero is preserved", func(t *testing.T) {
		zero := 0
		if got := resolveSetScheduleCapacity(&zero); got != 0 {
			t.Fatalf("got %d, want 0", got)
		}
	})

	t.Run("explicit value is preserved", func(t *testing.T) {
		five := 5
		if got := resolveSetScheduleCapacity(&five); got != 5 {
			t.Fatalf("got %d, want 5", got)
		}
	})
}

func TestParseYearMonthQuery(t *testing.T) {
	t.Run("accepts valid query", func(t *testing.T) {
		year, month, details := parseYearMonthQuery(newQueryRequest(t, url.Values{
			"year":  {"2026"},
			"month": {"2"},
		}))
		if len(details) != 0 {
			t.Fatalf("details = %+v, want none", details)
		}
		if year != 2026 || month != 2 {
			t.Fatalf("year=%d month=%d, want 2026 and 2", year, month)
		}
	})

	t.Run("missing or empty query", func(t *testing.T) {
		tests := []struct {
			name       string
			query      url.Values
			wantFields []string
		}{
			{name: "no query string", query: nil, wantFields: []string{"year", "month"}},
			{name: "empty year value", query: url.Values{"year": {""}, "month": {"2"}}, wantFields: []string{"year"}},
			{name: "empty month value", query: url.Values{"year": {"2026"}, "month": {""}}, wantFields: []string{"month"}},
			{name: "both empty values", query: url.Values{"year": {""}, "month": {""}}, wantFields: []string{"year", "month"}},
			{name: "whitespace only year", query: url.Values{"year": {" "}, "month": {"2"}}, wantFields: []string{"year"}},
			{name: "whitespace only month", query: url.Values{"year": {"2026"}, "month": {"\t"}}, wantFields: []string{"month"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, _, details := parseYearMonthQuery(newQueryRequest(t, tt.query))
				assertViolationFields(t, details, tt.wantFields)
			})
		}
	})

	t.Run("only one query parameter present", func(t *testing.T) {
		tests := []struct {
			name       string
			query      url.Values
			wantFields []string
		}{
			{name: "year only", query: url.Values{"year": {"2026"}}, wantFields: []string{"month"}},
			{name: "month only", query: url.Values{"month": {"3"}}, wantFields: []string{"year"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, _, details := parseYearMonthQuery(newQueryRequest(t, tt.query))
				assertViolationFields(t, details, tt.wantFields)
			})
		}
	})

	t.Run("non-numeric values", func(t *testing.T) {
		tests := []struct {
			name       string
			query      url.Values
			wantFields []string
		}{
			{name: "alphabetic year", query: url.Values{"year": {"abc"}, "month": {"2"}}, wantFields: []string{"year"}},
			{name: "alphabetic month", query: url.Values{"year": {"2026"}, "month": {"xx"}}, wantFields: []string{"month"}},
			{name: "both non-numeric", query: url.Values{"year": {"foo"}, "month": {"bar"}}, wantFields: []string{"year", "month"}},
			{name: "float year", query: url.Values{"year": {"2026.5"}, "month": {"2"}}, wantFields: []string{"year"}},
			{name: "scientific notation year", query: url.Values{"year": {"2e3"}, "month": {"2"}}, wantFields: []string{"year"}},
			// url.Values.Encode() 経由なら + は %2B になり strconv.Atoi("+2026") が成功する
			{name: "signed year with encoded plus", query: url.Values{"year": {"+2026"}, "month": {"2"}}, wantFields: nil},
			{name: "fullwidth digits year", query: url.Values{"year": {"２０２６"}, "month": {"2"}}, wantFields: []string{"year"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				year, month, details := parseYearMonthQuery(newQueryRequest(t, tt.query))
				if tt.wantFields == nil {
					if len(details) != 0 {
						t.Fatalf("details = %+v, want none (parsed year=%d month=%d)", details, year, month)
					}
					return
				}
				assertViolationFields(t, details, tt.wantFields)
			})
		}
	})

	// 特殊文字を含む値は rawQuery 直書きではなく url.Values + Encode すること
	// （未エスケープの ' 等は httptest.NewRequest が URL パース不能で panic の原因）
	t.Run("malicious or abusive query strings", func(t *testing.T) {
		tests := []struct {
			name       string
			query      url.Values
			rawQuery   string // 意図的にブラウザ寄りの生クエリを再現するときのみ使用
			wantFields []string
		}{
			{
				name:       "sql injection style year",
				query:      url.Values{"year": {"' OR 1=1 --"}, "month": {"2"}},
				wantFields: []string{"year"},
			},
			{
				name:       "script-like year value",
				query:      url.Values{"year": {"<script>2026</script>"}, "month": {"2"}},
				wantFields: []string{"year"},
			},
			{
				name: "very long year string",
				query: url.Values{
					"year":  {strings.Repeat("9", 64)},
					"month": {"2"},
				},
				wantFields: []string{"year"},
			},
			{
				name:       "duplicate year keys uses first value",
				rawQuery:   "year=2026&year=1999&month=2",
				wantFields: nil,
			},
			{
				name:       "duplicate month keys uses first value",
				rawQuery:   "year=2026&month=xx&month=2",
				wantFields: []string{"month"},
			},
			{
				name:       "null byte in year value",
				query:      url.Values{"year": {"2026\x00"}, "month": {"2"}},
				wantFields: []string{"year"},
			},
			{
				name:       "path traversal-like year",
				query:      url.Values{"year": {"../../etc/passwd"}, "month": {"2"}},
				wantFields: []string{"year"},
			},
			{
				// application/x-www-form-urlencoded では + がスペースにデコードされる
				// year=+2026 → TrimSpace 後 "2026" として受理（数値エラーにはならない）
				name:       "plus sign in form query decodes as space then trim accepts year",
				rawQuery:   "year=+2026&month=2",
				wantFields: nil,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var r *http.Request
				if tt.rawQuery != "" {
					r = httptest.NewRequest(http.MethodGet, "/?"+tt.rawQuery, nil)
				} else {
					r = newQueryRequest(t, tt.query)
				}
				year, month, details := parseYearMonthQuery(r)
				if tt.wantFields == nil {
					if len(details) != 0 {
						t.Fatalf("details = %+v, want none (parsed year=%d month=%d)", details, year, month)
					}
					return
				}
				assertViolationFields(t, details, tt.wantFields)
			})
		}
	})
}

// newQueryRequest はクエリを url.Values.Encode で組み立てる
// 生文字列の "?year=' OR ..." は URL パース不能や +→スペース変換で意図とずれるため使わない
func newQueryRequest(t *testing.T, query url.Values) *http.Request {
	t.Helper()
	target := "/"
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	return httptest.NewRequest(http.MethodGet, target, nil)
}

func assertViolationFields(t *testing.T, details []ErrorDetail, wantFields []string) {
	t.Helper()
	if len(details) != len(wantFields) {
		t.Fatalf("len(details)=%d, want %d: %+v", len(details), len(wantFields), details)
	}
	gotFields := make([]string, len(details))
	for i, d := range details {
		gotFields[i] = d.Field
	}
	if !slices.Equal(gotFields, wantFields) {
		t.Fatalf("fields = %v, want %v (details=%+v)", gotFields, wantFields, details)
	}
}
