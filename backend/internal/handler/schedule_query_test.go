package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

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
	// （未エスケープの ' 等は httptest.NewRequest が URL として解釈できず panic する）
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
				// year=+2026 → " 2026" となり Atoi 失敗（誤って year=2026 と期待するとテストが落ちる）
				name:       "plus sign in form query becomes space and fails atoi",
				rawQuery:   "year=+2026&month=2",
				wantFields: []string{"year"},
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
// 生文字列の "?year=' OR ..." は URL パースエラーや +→スペース変換で意図とずれるため使わない
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
