package usecase

import "testing"

func TestValidateListSchedulesYearMonth(t *testing.T) {
	tests := []struct {
		name      string
		year      int
		month     int
		wantField string
	}{
		{name: "accepts valid year and month", year: 2026, month: 2, wantField: ""},
		{name: "rejects year below range", year: 1999, month: 1, wantField: "year"},
		{name: "rejects month above range", year: 2026, month: 13, wantField: "month"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateListSchedulesYearMonth(tt.year, tt.month)
			if tt.wantField == "" {
				if len(got) != 0 {
					t.Fatalf("violations = %+v, want none", got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatal("violations empty, want at least one")
			}
			if got[0].Field != tt.wantField {
				t.Fatalf("Field = %q, want %q", got[0].Field, tt.wantField)
			}
		})
	}
}
