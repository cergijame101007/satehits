package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
)

func TestVisitDateAdvisoryKeys(t *testing.T) {
	tests := []struct {
		name       string
		date       datetime.Date
		wantNS     int32
		wantYYYYMM int32
	}{
		{
			name:       "maps 2026-08-21 to namespace 1 and key 20260821",
			date:       datetime.MustParseDate("2026-08-21"),
			wantNS:     1,
			wantYYYYMM: 20260821,
		},
		{
			name:       "maps 2025-01-01 to namespace 1 and key 20250101",
			date:       datetime.MustParseDate("2025-01-01"),
			wantNS:     1,
			wantYYYYMM: 20250101,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNS, gotKey := visitDateAdvisoryKeys(tt.date)
			if gotNS != tt.wantNS {
				t.Fatalf("namespace = %d, want %d", gotNS, tt.wantNS)
			}
			if gotKey != tt.wantYYYYMM {
				t.Fatalf("key = %d, want %d", gotKey, tt.wantYYYYMM)
			}
		})
	}
}

func TestPostgresVisitDateLocker_Lock_requiresTransaction(t *testing.T) {
	err := (PostgresVisitDateLocker{}).Lock(context.Background(), datetime.MustParseDate("2026-08-21"))
	if err == nil {
		t.Fatal("Lock() err = nil, want error")
	}
	if !strings.Contains(err.Error(), "requires a transaction") {
		t.Fatalf("Lock() err = %v, want requires a transaction", err)
	}
}

type recordingDBTX struct {
	query string
	args  []any
	calls int
}

func (r *recordingDBTX) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	r.calls++
	r.query = query
	r.args = append([]any(nil), args...)
	return nil, nil
}

func (r *recordingDBTX) QueryRowContext(context.Context, string, ...any) *sql.Row {
	panic("not implemented")
}

func (r *recordingDBTX) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("not implemented")
}

func TestExecVisitDateLock(t *testing.T) {
	db := &recordingDBTX{}
	date := datetime.MustParseDate("2026-08-21")

	if err := execVisitDateLock(context.Background(), db, date); err != nil {
		t.Fatalf("execVisitDateLock() err = %v, want nil", err)
	}
	if db.calls != 1 {
		t.Fatalf("ExecContext calls = %d, want 1", db.calls)
	}
	if !strings.Contains(db.query, "pg_advisory_xact_lock") {
		t.Fatalf("query = %q, want pg_advisory_xact_lock", db.query)
	}
	if len(db.args) != 2 {
		t.Fatalf("args len = %d, want 2", len(db.args))
	}
	ns, ok := db.args[0].(int32)
	if !ok || ns != 1 {
		t.Fatalf("args[0] = %v (%T), want int32(1)", db.args[0], db.args[0])
	}
	key, ok := db.args[1].(int32)
	if !ok || key != 20260821 {
		t.Fatalf("args[1] = %v (%T), want int32(20260821)", db.args[1], db.args[1])
	}
}
