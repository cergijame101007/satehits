package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

type fakePendingReminderReader struct {
	reservations []domain.Reservation
	listErr      error
	countErr     error
	totalPending int
	listFrom     datetime.Date
	listTo       datetime.Date
	countCalls   int
}

func (f *fakePendingReminderReader) ListPendingWebByVisitDateRange(_ context.Context, from, to datetime.Date) ([]domain.Reservation, error) {
	f.listFrom, f.listTo = from, to
	if f.listErr != nil {
		return nil, f.listErr
	}
	// 実 DB と同じく範囲で絞る（ウィンドウ判定が reader 側と usecase 側で二重に効くことを確認するため）
	var out []domain.Reservation
	for _, r := range f.reservations {
		if !r.VisitDate.Before(from.Time) && !r.VisitDate.After(to.Time) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakePendingReminderReader) CountPendingFrom(context.Context, datetime.Date) (int, error) {
	f.countCalls++
	if f.countErr != nil {
		return 0, f.countErr
	}
	return f.totalPending, nil
}

type enqueuedReminder struct {
	reservationID uuid.UUID
	mailType      domain.MailType
	totalPending  int
}

type fakePendingReminderEnqueuer struct {
	calls []enqueuedReminder
	// errByReservation は予約 ID ごとに返すエラー（未登録なら nil）
	errByReservation map[uuid.UUID]error
}

func (f *fakePendingReminderEnqueuer) EnqueuePendingReminder(_ context.Context, r domain.Reservation, mailType domain.MailType, totalPending int) error {
	f.calls = append(f.calls, enqueuedReminder{reservationID: r.ID, mailType: mailType, totalPending: totalPending})
	return f.errByReservation[r.ID]
}

func pendingWebReservation(id string, visitDate string) domain.Reservation {
	return domain.Reservation{
		ID:        uuid.MustParse(id),
		Name:      "山田太郎",
		People:    2,
		VisitDate: datetime.MustParseDate(visitDate),
		VisitTime: datetime.MustParseTime("11:30"),
		Status:    "pending",
		Source:    "web",
	}
}

func newPendingReminderUseCase(reader *fakePendingReminderReader, enqueuer *fakePendingReminderEnqueuer, now time.Time) *EnqueuePendingRemindersUseCase {
	uc := NewEnqueuePendingRemindersUseCase(reader, enqueuer)
	uc.now = func() time.Time { return now }
	return uc
}

// 2026-09-14 09:00 JST = 2026-09-14 00:00 UTC（店舗の今日は 2026-09-14）
var reminderNow = time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)

const (
	reservationA = "11111111-1111-1111-1111-111111111111"
	reservationB = "22222222-2222-2222-2222-222222222222"
	reservationC = "33333333-3333-3333-3333-333333333333"
)

func TestEnqueuePendingRemindersWindow(t *testing.T) {
	tests := []struct {
		name         string
		visitDate    string
		wantMailType domain.MailType
		wantEnqueued bool
	}{
		{name: "visit today gets 1d reminder", visitDate: "2026-09-14", wantMailType: domain.MailTypePendingReminder1D, wantEnqueued: true},
		{name: "visit tomorrow gets 1d reminder", visitDate: "2026-09-15", wantMailType: domain.MailTypePendingReminder1D, wantEnqueued: true},
		{name: "visit in 2 days gets 3d reminder", visitDate: "2026-09-16", wantMailType: domain.MailTypePendingReminder3D, wantEnqueued: true},
		{name: "visit in 3 days gets 3d reminder", visitDate: "2026-09-17", wantMailType: domain.MailTypePendingReminder3D, wantEnqueued: true},
		{name: "visit in 4 days is out of window", visitDate: "2026-09-18", wantEnqueued: false},
		{name: "visit yesterday is out of window", visitDate: "2026-09-13", wantEnqueued: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakePendingReminderReader{
				reservations: []domain.Reservation{pendingWebReservation(reservationA, tt.visitDate)},
				totalPending: 1,
			}
			enqueuer := &fakePendingReminderEnqueuer{}
			uc := newPendingReminderUseCase(reader, enqueuer, reminderNow)

			result, err := uc.Execute(context.Background())
			if err != nil {
				t.Fatalf("Execute() err = %v", err)
			}
			if !tt.wantEnqueued {
				if result.Candidates != 0 || len(enqueuer.calls) != 0 {
					t.Fatalf("result = %+v, calls = %d, want no candidates", result, len(enqueuer.calls))
				}
				return
			}
			if result.Candidates != 1 || result.Enqueued != 1 || result.Skipped != 0 {
				t.Fatalf("result = %+v, want candidates=1 enqueued=1 skipped=0", result)
			}
			if len(enqueuer.calls) != 1 || enqueuer.calls[0].mailType != tt.wantMailType {
				t.Fatalf("calls = %+v, want one call with %s", enqueuer.calls, tt.wantMailType)
			}
		})
	}
}

func TestEnqueuePendingRemindersQueriesWindowFromStoreToday(t *testing.T) {
	reader := &fakePendingReminderReader{}
	uc := newPendingReminderUseCase(reader, &fakePendingReminderEnqueuer{}, reminderNow)

	if _, err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if reader.listFrom.String() != "2026-09-14" || reader.listTo.String() != "2026-09-17" {
		t.Fatalf("list range = [%s, %s], want [2026-09-14, 2026-09-17]", reader.listFrom, reader.listTo)
	}
}

func TestEnqueuePendingRemindersUsesStoreCalendarDate(t *testing.T) {
	// UTC 23:30 は JST では翌日 08:30 なので、店舗の今日は 2026-09-15
	now := time.Date(2026, 9, 14, 23, 30, 0, 0, time.UTC)
	reader := &fakePendingReminderReader{
		reservations: []domain.Reservation{pendingWebReservation(reservationA, "2026-09-16")},
		totalPending: 1,
	}
	enqueuer := &fakePendingReminderEnqueuer{}
	uc := newPendingReminderUseCase(reader, enqueuer, now)

	if _, err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if reader.listFrom.String() != "2026-09-15" {
		t.Fatalf("list from = %s, want 2026-09-15 (JST date)", reader.listFrom)
	}
	// 2026-09-16 は JST 基準では翌日なので 1d、UTC の日付（09-14）基準なら 2 日後で 3d になってしまう
	if len(enqueuer.calls) != 1 || enqueuer.calls[0].mailType != domain.MailTypePendingReminder1D {
		t.Fatalf("calls = %+v, want one 1d reminder", enqueuer.calls)
	}
}

func TestEnqueuePendingRemindersSkipsAlreadyEnqueued(t *testing.T) {
	a := pendingWebReservation(reservationA, "2026-09-15")
	b := pendingWebReservation(reservationB, "2026-09-16")
	c := pendingWebReservation(reservationC, "2026-09-17")
	reader := &fakePendingReminderReader{reservations: []domain.Reservation{a, b, c}, totalPending: 3}
	enqueuer := &fakePendingReminderEnqueuer{
		errByReservation: map[uuid.UUID]error{b.ID: domain.ErrMailAlreadyEnqueued},
	}
	uc := newPendingReminderUseCase(reader, enqueuer, reminderNow)

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if result.Candidates != 3 || result.Enqueued != 2 || result.Skipped != 1 {
		t.Fatalf("result = %+v, want candidates=3 enqueued=2 skipped=1", result)
	}
	if len(enqueuer.calls) != 3 {
		t.Fatalf("calls = %d, want 3 (continues after already enqueued)", len(enqueuer.calls))
	}
}

func TestEnqueuePendingRemindersStopsOnEnqueueError(t *testing.T) {
	a := pendingWebReservation(reservationA, "2026-09-15")
	b := pendingWebReservation(reservationB, "2026-09-16")
	c := pendingWebReservation(reservationC, "2026-09-17")
	reader := &fakePendingReminderReader{reservations: []domain.Reservation{a, b, c}, totalPending: 3}
	boom := errors.New("db down")
	enqueuer := &fakePendingReminderEnqueuer{errByReservation: map[uuid.UUID]error{b.ID: boom}}
	uc := newPendingReminderUseCase(reader, enqueuer, reminderNow)

	result, err := uc.Execute(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want wrapped db error", err)
	}
	if len(enqueuer.calls) != 2 {
		t.Fatalf("calls = %d, want 2 (stops at the failing reservation)", len(enqueuer.calls))
	}
	if result.Enqueued != 1 {
		t.Fatalf("result = %+v, want enqueued=1 before failure", result)
	}
}

func TestEnqueuePendingRemindersReturnsReaderErrors(t *testing.T) {
	tests := []struct {
		name   string
		reader *fakePendingReminderReader
	}{
		{
			name:   "list failure",
			reader: &fakePendingReminderReader{listErr: errors.New("list failed")},
		},
		{
			name: "count failure",
			reader: &fakePendingReminderReader{
				reservations: []domain.Reservation{pendingWebReservation(reservationA, "2026-09-15")},
				countErr:     errors.New("count failed"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueuer := &fakePendingReminderEnqueuer{}
			uc := newPendingReminderUseCase(tt.reader, enqueuer, reminderNow)

			if _, err := uc.Execute(context.Background()); err == nil {
				t.Fatal("err = nil, want error")
			}
			if len(enqueuer.calls) != 0 {
				t.Fatalf("calls = %d, want 0", len(enqueuer.calls))
			}
		})
	}
}

func TestEnqueuePendingRemindersPassesTotalPending(t *testing.T) {
	reader := &fakePendingReminderReader{
		reservations: []domain.Reservation{pendingWebReservation(reservationA, "2026-09-15")},
		totalPending: 7,
	}
	enqueuer := &fakePendingReminderEnqueuer{}
	uc := newPendingReminderUseCase(reader, enqueuer, reminderNow)

	if _, err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(enqueuer.calls) != 1 || enqueuer.calls[0].totalPending != 7 {
		t.Fatalf("calls = %+v, want totalPending=7", enqueuer.calls)
	}
}

func TestEnqueuePendingRemindersWithNoCandidates(t *testing.T) {
	reader := &fakePendingReminderReader{totalPending: 5}
	enqueuer := &fakePendingReminderEnqueuer{}
	uc := newPendingReminderUseCase(reader, enqueuer, reminderNow)

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if result != (EnqueuePendingRemindersResult{}) {
		t.Fatalf("result = %+v, want all zero", result)
	}
	if reader.countCalls != 0 {
		t.Fatalf("count calls = %d, want 0 (no count without candidates)", reader.countCalls)
	}
	if len(enqueuer.calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(enqueuer.calls))
	}
}
