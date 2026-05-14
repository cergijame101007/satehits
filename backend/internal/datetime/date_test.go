package datetime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid date", input: "2026-02-10", want: "2026-02-10"},
		{name: "invalid text", input: "not-a-date", wantErr: true},
		{name: "invalid month", input: "2026-13-01", wantErr: true},
		{name: "invalid day", input: "2026-02-30", wantErr: true},
		{name: "date time is invalid", input: "2026-02-10T00:00:00Z", wantErr: true},
		{name: "empty string is invalid", input: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ParseDate(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDate: %v", err)
			}
			if got := d.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMustParseDate(t *testing.T) {
	d := MustParseDate("2026-05-01")
	if d.String() != "2026-05-01" {
		t.Fatalf("got %q", d.String())
	}
}

func TestNewDate(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month time.Month
		day   int
		want  string
	}{
		{name: "normal date", year: 2026, month: time.February, day: 28, want: "2026-02-28"},
		{name: "leap day", year: 2024, month: time.February, day: 29, want: "2024-02-29"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDate(tc.year, tc.month, tc.day)
			if got := d.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDate_JSONRoundTrip(t *testing.T) {
	type row struct {
		D Date `json:"d"`
	}
	in := row{D: MustParseDate("2026-02-10")}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out row
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.D.String() != in.D.String() {
		t.Fatalf("got %q, want %q", out.D.String(), in.D.String())
	}
}

func TestDate_JSONNull(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "null", input: `null`},
		{name: "empty string", input: `""`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var d Date
			if err := json.Unmarshal([]byte(tc.input), &d); err != nil {
				t.Fatal(err)
			}
			if !d.IsZero() {
				t.Fatalf("expected zero Date, got %v", d.Time)
			}
		})
	}
}

func TestDate_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{
			name:  "time.Time keeps local date fields",
			input: time.Date(2026, 3, 15, 14, 30, 0, 0, time.FixedZone("JST", 9*3600)),
			want:  "2026-03-15",
		},
		{name: "zero time.Time", input: time.Time{}, want: ""},
		{name: "byte slice", input: []byte("2026-01-02"), want: "2026-01-02"},
		{name: "string", input: "2025-12-31", want: "2025-12-31"},
		{name: "nil", input: nil, want: ""},
		{name: "empty string", input: "", want: ""},
		{name: "invalid string", input: "2026-02-30", wantErr: true},
		{name: "date time string is invalid", input: "2026-02-10T00:00:00Z", wantErr: true},
		{name: "unsupported int", input: 123, wantErr: true},
		{name: "unsupported bool", input: true, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var d Date
			err := d.Scan(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if got := d.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDate_Value(t *testing.T) {
	tests := []struct {
		name string
		date Date
		want any
	}{
		{name: "valid date", date: MustParseDate("2026-02-10"), want: "2026-02-10"},
		{name: "zero date", date: Date{}, want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.date.Value()
			if err != nil {
				t.Fatalf("Value: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Value() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestDate_AddDays(t *testing.T) {
	tests := []struct {
		name string
		date Date
		days int
		want string
	}{
		{name: "next day", date: MustParseDate("2026-02-01"), days: 1, want: "2026-02-02"},
		{name: "fourteen days", date: MustParseDate("2026-02-01"), days: 14, want: "2026-02-15"},
		{name: "previous day", date: MustParseDate("2026-02-01"), days: -1, want: "2026-01-31"},
		{name: "month boundary", date: MustParseDate("2026-01-31"), days: 1, want: "2026-02-01"},
		{name: "leap day boundary", date: MustParseDate("2024-02-28"), days: 1, want: "2024-02-29"},
		{name: "zero date stays zero", date: Date{}, days: 5, want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.date.AddDays(tc.days)
			if got.String() != tc.want {
				t.Fatalf("AddDays(%d) = %q, want %q", tc.days, got.String(), tc.want)
			}
		})
	}
}
