package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cergijame101007/satehits/internal/domain"
)

const (
	mailTypeReceived = "reservation_received"
	mailTypeApproved = "reservation_approved"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	applyAllMigrations(t, db)
	return db
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}

func applyAllMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := db.Exec(string(body)); err != nil {
			// 再実行時は CREATE 済みでも可（IF NOT EXISTS）。致命的エラーのみ落とす
			if !isIgnorableMigrationError(err) {
				t.Fatalf("apply %s: %v", name, err)
			}
		}
	}
}

func isIgnorableMigrationError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "already exists")
}

func truncateOutboxTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`TRUNCATE email_outbox, reservations RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// insertReservation は予約を 1 件作る。同一日時 + 電話番号の active 予約は UNIQUE なので電話番号を都度変える
func insertReservation(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	phone := "090-" + uuid.NewString()[:8]
	err := db.QueryRow(`
INSERT INTO reservations (name, people, visit_date, visit_time, phone, email, status, source)
VALUES ('テスト', 2, '2030-01-15', '12:00', $1, 't@example.com', 'pending', 'web')
RETURNING id`, phone).Scan(&id)
	if err != nil {
		t.Fatalf("insert reservation: %v", err)
	}
	return id
}

func insertOutboxRow(t *testing.T, db *sql.DB, reservationID uuid.UUID, mailType string, createdAt time.Time) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(`
INSERT INTO email_outbox (
    reservation_id, mail_type, from_address, to_address, subject, body_html, body_text, created_at
) VALUES ($1, $2, 'from@example.com', 'to@example.com', 'subj', '<p>x</p>', 'x', $3)
RETURNING id`, reservationID, mailType, createdAt).Scan(&id)
	if err != nil {
		t.Fatalf("insert outbox: %v", err)
	}
	return id
}

type outboxRowState struct {
	status       string
	attemptCount int
	lastError    sql.NullString
}

func readOutboxRow(t *testing.T, db *sql.DB, id uuid.UUID) outboxRowState {
	t.Helper()
	var st outboxRowState
	err := db.QueryRow(`SELECT status, attempt_count, last_error FROM email_outbox WHERE id = $1`, id).
		Scan(&st.status, &st.attemptCount, &st.lastError)
	if err != nil {
		t.Fatalf("read outbox row: %v", err)
	}
	return st
}

func claim(t *testing.T, repo *PostgresEmailOutboxRepository, leaseUntil time.Time) *domain.EmailOutboxMessage {
	t.Helper()
	msg, err := repo.ClaimNextPending(context.Background(), leaseUntil)
	if err != nil {
		t.Fatalf("ClaimNextPending: %v", err)
	}
	return msg
}

func TestEmailOutboxEnqueueDuplicateReturnsAlreadyEnqueued(t *testing.T) {
	db := openTestDB(t)
	truncateOutboxTables(t, db)
	repo := NewPostgresEmailOutboxRepository(db)
	reservationID := insertReservation(t, db)

	in := domain.EnqueueEmailInput{
		ReservationID: reservationID,
		MailType:      domain.MailTypeReservationReceived,
		FromAddress:   "from@example.com",
		ToAddress:     "to@example.com",
		Subject:       "subj",
		BodyHTML:      "<p>x</p>",
		BodyText:      "x",
	}
	if err := repo.Enqueue(context.Background(), in); err != nil {
		t.Fatalf("first Enqueue: %v", err)
	}
	err := repo.Enqueue(context.Background(), in)
	if !errors.Is(err, domain.ErrMailAlreadyEnqueued) {
		t.Fatalf("second Enqueue err = %v, want ErrMailAlreadyEnqueued", err)
	}
}

func TestEmailOutboxClaim(t *testing.T) {
	db := openTestDB(t)
	repo := NewPostgresEmailOutboxRepository(db)
	t0 := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	future := time.Now().Add(5 * time.Minute)
	past := time.Now().Add(-time.Minute)

	t.Run("increments attempt count and holds lease until it expires", func(t *testing.T) {
		truncateOutboxTables(t, db)
		reservationID := insertReservation(t, db)
		id := insertOutboxRow(t, db, reservationID, mailTypeReceived, t0)

		first := claim(t, repo, future)
		if first == nil || first.ID != id || first.AttemptCount != 1 {
			t.Fatalf("first claim = %+v, want id=%s attempt=1", first, id)
		}
		if again := claim(t, repo, future); again != nil {
			t.Fatalf("claim during lease = %+v, want nil", again)
		}

		// lease 切れ（過去の時刻で claim し直す）→ 同じ行が attempt 2 で取れる
		if _, err := db.Exec(`UPDATE email_outbox SET next_attempt_at = $2 WHERE id = $1`, id, past); err != nil {
			t.Fatalf("expire lease: %v", err)
		}
		second := claim(t, repo, future)
		if second == nil || second.ID != id || second.AttemptCount != 2 {
			t.Fatalf("claim after lease expiry = %+v, want id=%s attempt=2", second, id)
		}
	})

	t.Run("does not claim later row while earlier pending row of same reservation exists", func(t *testing.T) {
		truncateOutboxTables(t, db)
		reservationID := insertReservation(t, db)
		idA := insertOutboxRow(t, db, reservationID, mailTypeReceived, t0)
		idB := insertOutboxRow(t, db, reservationID, mailTypeApproved, t0.Add(time.Minute))

		msgA := claim(t, repo, future)
		if msgA == nil || msgA.ID != idA {
			t.Fatalf("claimed = %+v, want idA=%s", msgA, idA)
		}
		// A は lease 中（pending のまま）なので B も取れない
		if msgB := claim(t, repo, future); msgB != nil {
			t.Fatalf("claimed = %+v, want nil while A is leased", msgB)
		}

		if err := repo.MarkSent(context.Background(), idA); err != nil {
			t.Fatalf("MarkSent A: %v", err)
		}
		after := claim(t, repo, future)
		if after == nil || after.ID != idB {
			t.Fatalf("claimed after A sent = %+v, want idB=%s", after, idB)
		}
	})

	t.Run("claims later row when earlier row of same reservation is failed", func(t *testing.T) {
		truncateOutboxTables(t, db)
		reservationID := insertReservation(t, db)
		idA := insertOutboxRow(t, db, reservationID, mailTypeReceived, t0)
		idB := insertOutboxRow(t, db, reservationID, mailTypeApproved, t0.Add(time.Minute))

		if msgA := claim(t, repo, future); msgA == nil || msgA.ID != idA {
			t.Fatalf("claimed = %+v, want idA=%s", msgA, idA)
		}
		if err := repo.MarkFailed(context.Background(), idA, "boom"); err != nil {
			t.Fatalf("MarkFailed A: %v", err)
		}
		if got := readOutboxRow(t, db, idA); got.status != domain.OutboxStatusFailed || got.attemptCount != 1 || got.lastError.String != "boom" {
			t.Fatalf("row A = %+v, want failed attempt=1 lastError=boom", got)
		}
		after := claim(t, repo, future)
		if after == nil || after.ID != idB {
			t.Fatalf("claimed after A failed = %+v, want idB=%s", after, idB)
		}
	})

	t.Run("claims rows of different reservations in created order", func(t *testing.T) {
		truncateOutboxTables(t, db)
		r1 := insertReservation(t, db)
		r2 := insertReservation(t, db)
		idLater := insertOutboxRow(t, db, r1, mailTypeReceived, t0.Add(time.Minute))
		idEarlier := insertOutboxRow(t, db, r2, mailTypeReceived, t0)

		first := claim(t, repo, future)
		second := claim(t, repo, future)
		if first == nil || second == nil || first.ID != idEarlier || second.ID != idLater {
			t.Fatalf("claim order = (%v, %v), want (%s, %s)", first, second, idEarlier, idLater)
		}
	})
}

func TestEmailOutboxMarks(t *testing.T) {
	db := openTestDB(t)
	repo := NewPostgresEmailOutboxRepository(db)
	t0 := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	future := time.Now().Add(5 * time.Minute)
	past := time.Now().Add(-time.Minute)

	t.Run("retry keeps attempt count and reschedules", func(t *testing.T) {
		truncateOutboxTables(t, db)
		id := insertOutboxRow(t, db, insertReservation(t, db), mailTypeReceived, t0)
		claim(t, repo, future)

		if err := repo.MarkRetry(context.Background(), id, past, "temporary"); err != nil {
			t.Fatalf("MarkRetry: %v", err)
		}
		got := readOutboxRow(t, db, id)
		if got.status != domain.OutboxStatusPending || got.attemptCount != 1 || got.lastError.String != "temporary" {
			t.Fatalf("row = %+v, want pending attempt=1 lastError=temporary", got)
		}
		if again := claim(t, repo, future); again == nil || again.AttemptCount != 2 {
			t.Fatalf("claim after retry = %+v, want attempt=2", again)
		}
	})

	t.Run("release claim gives the attempt back", func(t *testing.T) {
		truncateOutboxTables(t, db)
		id := insertOutboxRow(t, db, insertReservation(t, db), mailTypeReceived, t0)
		claim(t, repo, future)

		if err := repo.ReleaseClaim(context.Background(), id, past, "auth"); err != nil {
			t.Fatalf("ReleaseClaim: %v", err)
		}
		got := readOutboxRow(t, db, id)
		if got.status != domain.OutboxStatusPending || got.attemptCount != 0 || got.lastError.String != "auth" {
			t.Fatalf("row = %+v, want pending attempt=0 lastError=auth", got)
		}
		if again := claim(t, repo, future); again == nil || again.AttemptCount != 1 {
			t.Fatalf("claim after release = %+v, want attempt=1", again)
		}
	})

	t.Run("sent clears last error and rejects further marks", func(t *testing.T) {
		truncateOutboxTables(t, db)
		id := insertOutboxRow(t, db, insertReservation(t, db), mailTypeReceived, t0)
		claim(t, repo, future)
		if err := repo.MarkRetry(context.Background(), id, future, "temporary"); err != nil {
			t.Fatalf("MarkRetry: %v", err)
		}

		if err := repo.MarkSent(context.Background(), id); err != nil {
			t.Fatalf("MarkSent: %v", err)
		}
		got := readOutboxRow(t, db, id)
		if got.status != domain.OutboxStatusSent || got.lastError.Valid {
			t.Fatalf("row = %+v, want sent with NULL last_error", got)
		}
		if err := repo.MarkFailed(context.Background(), id, "late"); err == nil {
			t.Fatal("MarkFailed on sent row err = nil, want error")
		}
		if err := repo.MarkSent(context.Background(), id); err == nil {
			t.Fatal("MarkSent twice err = nil, want error")
		}
	})
}
