package datetime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid time", input: "12:30", want: "12:30"},
		{name: "midnight", input: "00:00", want: "00:00"},
		{name: "last minute of day", input: "23:59", want: "23:59"},
		{name: "invalid hour", input: "25:00", wantErr: true},
		{name: "invalid minute", input: "12:60", wantErr: true},
		{name: "seconds are invalid for request format", input: "08:00:00", wantErr: true},
		{name: "empty string is invalid", input: "", wantErr: true},
		{name: "invalid text", input: "noon", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tm, err := ParseTime(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseTime(%q) err = nil, want error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTime(%q) err = %v, want nil", tc.input, err)
			}
			if got := tm.String(); got != tc.want {
				t.Fatalf("ParseTime(%q).String() = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestMustParseTime(t *testing.T) {
	tm := MustParseTime("09:05")
	if tm.String() != "09:05" {
		t.Fatalf("got %q", tm.String())
	}
}

func TestNewTime(t *testing.T) {
	tests := []struct {
		name   string
		hour   int
		minute int
		want   string
	}{
		{name: "normal time", hour: 18, minute: 45, want: "18:45"},
		{name: "midnight", hour: 0, minute: 0, want: "00:00"},
		{name: "last minute", hour: 23, minute: 59, want: "23:59"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tm := NewTime(tc.hour, tc.minute)
			if got := tm.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTime_JSONRoundTrip(t *testing.T) {
	type row struct {
		Clock Time `json:"t"`
	}
	in := row{Clock: MustParseTime("12:00")}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out row
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Clock.String() != in.Clock.String() {
		t.Fatalf("got %q, want %q", out.Clock.String(), in.Clock.String())
	}
}

func TestTime_JSONNull(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "null", input: `null`},
		{name: "empty string", input: `""`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var tm Time
			if err := json.Unmarshal([]byte(tc.input), &tm); err != nil {
				t.Fatal(err)
			}
			if !tm.IsZero() {
				t.Fatalf("expected zero Time, got %v", tm.Time)
			}
		})
	}
}

func TestTime_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{name: "time.Time", input: time.Date(2000, 1, 1, 14, 5, 30, 0, time.UTC), want: "14:05"},
		{name: "time.Time with non UTC location", input: time.Date(2000, 1, 1, 7, 30, 0, 0, time.FixedZone("JST", 9*3600)), want: "07:30"},
		{name: "zero time.Time", input: time.Time{}, want: ""},
		{name: "byte slice HH:MM", input: []byte("09:15"), want: "09:15"},
		{name: "byte slice HH:MM:SS", input: []byte("09:15:30"), want: "09:15"},
		{name: "string HH:MM", input: "08:00", want: "08:00"},
		{name: "string HH:MM:SS", input: "08:00:00", want: "08:00"},
		{name: "nil", input: nil, want: ""},
		{name: "empty string", input: "", want: ""},
		{name: "invalid hour", input: "25:00", wantErr: true},
		{name: "invalid minute", input: "12:60", wantErr: true},
		{name: "unsupported int", input: 123, wantErr: true},
		{name: "unsupported bool", input: false, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var tm Time
			err := tm.Scan(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if got := tm.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTime_Value(t *testing.T) {
	tests := []struct {
		name string
		tm   Time
		want any
	}{
		{name: "valid time", tm: MustParseTime("12:30"), want: "12:30:00"},
		{name: "time with seconds from Scan truncated", tm: mustScanTime(t, "09:15:30"), want: "09:15:00"},
		{name: "zero time", tm: Time{}, want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.tm.Value()
			if err != nil {
				t.Fatalf("Value: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Value() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func mustScanTime(t *testing.T, value any) Time {
	t.Helper()
	var tm Time
	if err := tm.Scan(value); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return tm
}
