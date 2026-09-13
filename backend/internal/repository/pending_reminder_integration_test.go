package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// insertReservationWith は status / source / 来店日を指定して予約を 1 件作る
func insertReservationWith(t *testing.T, db *sql.DB, visitDate, status, source string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	phone := "090-" + uuid.NewString()[:8]
	err := db.QueryRow(`
INSERT INTO reservations (name, people, visit_date, visit_time, phone, email, status, source)
VALUES ('テスト', 2, $1, '12:00', $2, 't@example.com', $3, $4)
RETURNING id`, visitDate, phone, status, source).Scan(&id)
	if err != nil {
		t.Fatalf("insert reservation: %v", err)
	}
	return id
}

func TestPendingReminderReaderListPendingWebByVisitDateRange(t *testing.T) {
	db := openTestDB(t)
	truncateOutboxTables(t, db)
	repo := NewPostgresReservationRepository(db)

	inRangeEarly := insertReservationWith(t, db, "2030-01-10", "pending", "web")
	inRangeLate := insertReservationWith(t, db, "2030-01-13", "pending", "web")
	insertReservationWith(t, db, "2030-01-09", "pending", "web")       // 範囲外（前日）
	insertReservationWith(t, db, "2030-01-14", "pending", "web")       // 範囲外（翌日）
	insertReservationWith(t, db, "2030-01-11", "approved", "web")      // status 不一致
	insertReservationWith(t, db, "2030-01-11", "pending", "instagram") // source 不一致

	got, err := repo.ListPendingWebByVisitDateRange(context.Background(),
		datetime.MustParseDate("2030-01-10"), datetime.MustParseDate("2030-01-13"))
	if err != nil {
		t.Fatalf("ListPendingWebByVisitDateRange: %v", err)
	}
	if len(got) != 2 || got[0].ID != inRangeEarly || got[1].ID != inRangeLate {
		t.Fatalf("got %d rows (%v), want [%s, %s] in visit_date order", len(got), got, inRangeEarly, inRangeLate)
	}
}

func TestPendingReminderReaderCountPendingFrom(t *testing.T) {
	db := openTestDB(t)
	truncateOutboxTables(t, db)
	repo := NewPostgresReservationRepository(db)

	insertReservationWith(t, db, "2030-01-10", "pending", "web")
	insertReservationWith(t, db, "2030-01-20", "pending", "instagram") // source を問わず数える
	insertReservationWith(t, db, "2030-01-09", "pending", "web")       // from より前は数えない
	insertReservationWith(t, db, "2030-01-15", "approved", "web")      // pending 以外は数えない

	got, err := repo.CountPendingFrom(context.Background(), datetime.MustParseDate("2030-01-10"))
	if err != nil {
		t.Fatalf("CountPendingFrom: %v", err)
	}
	if got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}

func TestEmailOutboxAcceptsPendingReminderMailTypes(t *testing.T) {
	db := openTestDB(t)
	truncateOutboxTables(t, db)
	repo := NewPostgresEmailOutboxRepository(db)
	reservationID := insertReservation(t, db)

	for _, mailType := range []domain.MailType{domain.MailTypePendingReminder3D, domain.MailTypePendingReminder1D} {
		t.Run("accepts "+string(mailType), func(t *testing.T) {
			err := repo.Enqueue(context.Background(), domain.EnqueueEmailInput{
				ReservationID: reservationID,
				MailType:      mailType,
				FromAddress:   "from@example.com",
				ToAddress:     "owner@example.com",
				Subject:       "subj",
				BodyHTML:      "<p>x</p>",
				BodyText:      "x",
			})
			if err != nil {
				t.Fatalf("Enqueue(%s): %v", mailType, err)
			}
		})
	}
}
