package repository

import (
	"database/sql"
	"testing"
)

func TestScanEventText(t *testing.T) {
	t.Run("null becomes empty string", func(t *testing.T) {
		if got := scanEventText(sql.NullString{}); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})

	t.Run("valid value is preserved", func(t *testing.T) {
		ns := sql.NullString{String: "和紅茶をしばく会", Valid: true}
		if got := scanEventText(ns); got != "和紅茶をしばく会" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestEventTextParam(t *testing.T) {
	t.Run("empty becomes null", func(t *testing.T) {
		p := eventTextParam("")
		if p.Valid {
			t.Fatal("expected invalid NullString")
		}
	})

	t.Run("whitespace only becomes null", func(t *testing.T) {
		p := eventTextParam("  \t ")
		if p.Valid {
			t.Fatal("expected invalid NullString")
		}
	})

	t.Run("non-empty is trimmed and valid", func(t *testing.T) {
		p := eventTextParam("  リゾット  ")
		if !p.Valid || p.String != "リゾット" {
			t.Fatalf("got Valid=%v String=%q", p.Valid, p.String)
		}
	})
}
