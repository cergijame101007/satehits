package holiday

import (
	"strings"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantLen      int
		wantLastYear int
		wantContains []string
		wantMissing  []string
	}{
		{
			name:         "parses header and rows in cabinet office format",
			input:        "国民の祝日・休日月日,国民の祝日・休日名称\n2026/1/1,元日\n2026/5/6,休日\n",
			wantLen:      2,
			wantLastYear: 2026,
			wantContains: []string{"2026-01-01", "2026-05-06"},
			wantMissing:  []string{"2026-01-02"},
		},
		{
			name:         "strips utf8 bom before header",
			input:        "\uFEFF国民の祝日・休日月日,国民の祝日・休日名称\n2026/2/11,建国記念の日\n",
			wantLen:      1,
			wantLastYear: 2026,
			wantContains: []string{"2026-02-11"},
		},
		{
			name:         "accepts crlf line endings",
			input:        "国民の祝日・休日月日,国民の祝日・休日名称\r\n2026/3/20,春分の日\r\n",
			wantLen:      1,
			wantLastYear: 2026,
			wantContains: []string{"2026-03-20"},
		},
		{
			name:         "skips blank and malformed rows",
			input:        "2026/1/1,元日\n\n   \nnot-a-date,x\n2026/13/1,invalid month\n,名称のみ\n2027/1/1\n",
			wantLen:      2,
			wantLastYear: 2027,
			wantContains: []string{"2026-01-01", "2027-01-01"},
		},
		{
			name:         "tracks the last year across unordered rows",
			input:        "2027/1/1,元日\n2025/1/1,元日\n2026/1/1,元日\n",
			wantLen:      3,
			wantLastYear: 2027,
		},
		{
			name:         "returns empty set for empty input",
			input:        "",
			wantLen:      0,
			wantLastYear: 0,
			wantMissing:  []string{"2026-01-01"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got.Len() != tt.wantLen {
				t.Fatalf("Len = %d, want %d", got.Len(), tt.wantLen)
			}
			if got.LastYear() != tt.wantLastYear {
				t.Fatalf("LastYear = %d, want %d", got.LastYear(), tt.wantLastYear)
			}
			for _, d := range tt.wantContains {
				if !got.IsNationalHoliday(datetime.MustParseDate(d)) {
					t.Fatalf("IsNationalHoliday(%s) = false, want true", d)
				}
			}
			for _, d := range tt.wantMissing {
				if got.IsNationalHoliday(datetime.MustParseDate(d)) {
					t.Fatalf("IsNationalHoliday(%s) = true, want false", d)
				}
			}
		})
	}
}

func TestSet_IsNationalHoliday_nilSafety(t *testing.T) {
	tests := []struct {
		name string
		set  *Set
		date datetime.Date
	}{
		{name: "nil set never contains", set: nil, date: datetime.MustParseDate("2026-01-01")},
		{name: "zero date is never a holiday", set: mustParse(t, "2026/1/1,元日\n"), date: datetime.Date{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set.IsNationalHoliday(tt.date) {
				t.Fatal("IsNationalHoliday = true, want false")
			}
		})
	}
}

func TestSet_NeedsUpdate(t *testing.T) {
	set := mustParse(t, "2026/1/1,元日\n2027/1/1,元日\n")
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "fresh while next year is bundled", now: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), want: false},
		{name: "stays quiet before march while next year is not yet published", now: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), want: false},
		{name: "stays quiet at the end of february", now: time.Date(2027, 2, 28, 23, 59, 0, 0, time.UTC), want: false},
		{name: "needs update from march once next year should be published", now: time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC), want: true},
		{name: "needs update when bundled data is older than the current year even in january", now: time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.NeedsUpdate(tt.now); got != tt.want {
				t.Fatalf("NeedsUpdate = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbedded(t *testing.T) {
	// 同梱 CSV の内容に依存する最低限のチェック（翌年分を取り込んだら年は進む）
	tests := []struct {
		name string
		date string
		want bool
	}{
		{name: "new year's day 2026 is a holiday", date: "2026-01-01", want: true},
		{name: "substitute holiday 2026-05-06 is a holiday", date: "2026-05-06", want: true},
		{name: "ordinary weekday is not a holiday", date: "2026-05-18", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Embedded().IsNationalHoliday(datetime.MustParseDate(tt.date)); got != tt.want {
				t.Fatalf("Embedded().IsNationalHoliday(%s) = %v, want %v", tt.date, got, tt.want)
			}
		})
	}
	if Embedded().LastYear() < 2026 {
		t.Fatalf("Embedded().LastYear() = %d, want >= 2026", Embedded().LastYear())
	}
}

func mustParse(t *testing.T, input string) *Set {
	t.Helper()
	s, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return s
}
