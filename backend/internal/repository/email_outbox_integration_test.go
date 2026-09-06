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

func insertReservation(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(`
INSERT INTO reservations (name, people, visit_date, visit_time, phone, email, status, source)
VALUES ('テスト', 2, '2030-01-15', '12:00', '090-0000-0001', 't@example.com', 'pending', 'web')
RETURNING id`).Scan(&id)
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

func TestEmailOutboxClaimSerializesByReservationID(t *testing.T) {
	db := openTestDB(t)
	truncateOutboxTables(t, db)
	repo := NewPostgresEmailOutboxRepository(db)
	reservationID := insertReservation(t, db)

	t0 := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	idA := insertOutboxRow(t, db, reservationID, "reservation_received", t0)
	idB := insertOutboxRow(t, db, reservationID, "reservation_approved", t0.Add(time.Minute))

	tx1, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}
	defer func() { _ = tx1.Rollback() }()
	ctx1 := withTx(context.Background(), tx1)

	msgA, err := repo.ClaimNextPending(ctx1)
	if err != nil {
		t.Fatalf("tx1 ClaimNextPending: %v", err)
	}
	if msgA == nil || msgA.ID != idA {
		t.Fatalf("tx1 claimed = %+v, want idA=%s", msgA, idA)
	}

	// tx2 は A を SKIP し、かつ先行 pending がある B も取らない
	done := make(chan struct{})
	var msgB *domain.EmailOutboxMessage
	var claimErr error
	go func() {
		defer close(done)
		tx2, err := db.Begin()
		if err != nil {
			claimErr = err
			return
		}
		defer func() { _ = tx2.Rollback() }()
		ctx2 := withTx(context.Background(), tx2)
		msgB, claimErr = repo.ClaimNextPending(ctx2)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("tx2 ClaimNextPending timed out")
	}
	if claimErr != nil {
		t.Fatalf("tx2 ClaimNextPending: %v", claimErr)
	}
	if msgB != nil {
		t.Fatalf("tx2 claimed = %+v, want nil while A is locked and pending", msgB)
	}

	if err := tx1.Commit(); err != nil {
		t.Fatalf("commit tx1: %v", err)
	}
	// A を sent にして先行 pending を解消する
	txMark, err := db.Begin()
	if err != nil {
		t.Fatalf("begin mark: %v", err)
	}
	if err := repo.MarkSent(withTx(context.Background(), txMark), idA); err != nil {
		_ = txMark.Rollback()
		t.Fatalf("MarkSent A: %v", err)
	}
	if err := txMark.Commit(); err != nil {
		t.Fatalf("commit mark: %v", err)
	}

	tx3, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx3: %v", err)
	}
	defer func() { _ = tx3.Rollback() }()
	msgAfter, err := repo.ClaimNextPending(withTx(context.Background(), tx3))
	if err != nil {
		t.Fatalf("tx3 ClaimNextPending: %v", err)
	}
	if msgAfter == nil || msgAfter.ID != idB {
		t.Fatalf("tx3 claimed = %+v, want idB=%s", msgAfter, idB)
	}
}
