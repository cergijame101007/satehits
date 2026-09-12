package usecase

import (
	"context"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/holiday"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

// testStoreCalendar は同梱の祝日データを使う StoreCalendar（このパッケージのテストは祝日の有無に依存しない）
func testStoreCalendar() *service.StoreCalendar {
	return service.NewStoreCalendar(holiday.Embedded())
}

// stubScheduleRepo は Upsert の記録用
type stubScheduleRepo struct {
	lastIn   domain.SetScheduleInput
	inserted bool
}

func (s *stubScheduleRepo) Upsert(ctx context.Context, in domain.SetScheduleInput) (domain.Schedule, bool, error) {
	s.lastIn = in
	return domain.Schedule{
		Date:             in.Date,
		ScheduleType:     in.ScheduleType,
		Capacity:         in.Capacity,
		EventName:        in.EventName,
		EventDescription: in.EventDescription,
		OpenTime:         in.OpenTime,
		LastOrderTime:    in.LastOrderTime,
		CloseTime:        in.CloseTime,
	}, s.inserted, nil
}

func (s *stubScheduleRepo) FindByDate(ctx context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (s *stubScheduleRepo) ListStoredByYearMonth(ctx context.Context, year, month int) ([]domain.Schedule, error) {
	return nil, nil
}

func (s *stubScheduleRepo) DeleteByDate(ctx context.Context, date datetime.Date) error {
	return nil
}

func TestSetScheduleUseCase_Execute_appliesDefaultBusinessHours(t *testing.T) {
	repo := &stubScheduleRepo{inserted: true}
	uc := NewSetScheduleUseCase(repo, testStoreCalendar())

	_, err := uc.Execute(context.Background(), SetScheduleCommand{
		Date:         datetime.MustParseDate("2026-05-20"),
		ScheduleType: "normal",
		Capacity:     10,
	})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if repo.lastIn.OpenTime.String() != "11:30" {
		t.Fatalf("OpenTime = %s, want 11:30", repo.lastIn.OpenTime.String())
	}
	if repo.lastIn.LastOrderTime.String() != "13:30" {
		t.Fatalf("LastOrderTime = %s, want 13:30", repo.lastIn.LastOrderTime.String())
	}
	if repo.lastIn.CloseTime.String() != "15:00" {
		t.Fatalf("CloseTime = %s, want 15:00", repo.lastIn.CloseTime.String())
	}
}

func TestSetScheduleUseCase_Execute_appliesWeekdayDefaultsForEvent(t *testing.T) {
	repo := &stubScheduleRepo{inserted: true}
	uc := NewSetScheduleUseCase(repo, testStoreCalendar())

	_, err := uc.Execute(context.Background(), SetScheduleCommand{
		Date:             datetime.MustParseDate("2026-05-20"),
		ScheduleType:     "event",
		Capacity:         10,
		EventName:        "和紅茶をしばく会",
		EventDescription: "和紅茶をしばく会 入門編",
	})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if repo.lastIn.OpenTime.String() != "11:30" {
		t.Fatalf("OpenTime = %s, want 11:30", repo.lastIn.OpenTime.String())
	}
	if repo.lastIn.LastOrderTime.String() != "13:30" {
		t.Fatalf("LastOrderTime = %s, want 13:30", repo.lastIn.LastOrderTime.String())
	}
	if repo.lastIn.CloseTime.String() != "15:00" {
		t.Fatalf("CloseTime = %s, want 15:00", repo.lastIn.CloseTime.String())
	}
}

func TestSetScheduleUseCase_Execute_clearsTimesForClosed(t *testing.T) {
	repo := &stubScheduleRepo{inserted: true}
	uc := NewSetScheduleUseCase(repo, testStoreCalendar())

	_, err := uc.Execute(context.Background(), SetScheduleCommand{
		Date:          datetime.MustParseDate("2026-05-21"),
		ScheduleType:  "closed",
		Capacity:      0,
		OpenTime:      datetime.MustParseTime("11:30"),
		LastOrderTime: datetime.MustParseTime("13:30"),
		CloseTime:     datetime.MustParseTime("15:00"),
	})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if !repo.lastIn.OpenTime.IsZero() || !repo.lastIn.LastOrderTime.IsZero() || !repo.lastIn.CloseTime.IsZero() {
		t.Fatalf("times = %s/%s/%s, want all zero for closed",
			repo.lastIn.OpenTime, repo.lastIn.LastOrderTime, repo.lastIn.CloseTime)
	}
}

func TestSetScheduleUseCase_Execute_clearsEventTextForNormalAndClosed(t *testing.T) {
	t.Run("clears omitted event text on closed", func(t *testing.T) {
		repo := &stubScheduleRepo{inserted: true}
		uc := NewSetScheduleUseCase(repo, testStoreCalendar())

		_, err := uc.Execute(context.Background(), SetScheduleCommand{
			Date:         datetime.MustParseDate("2026-05-21"),
			ScheduleType: "closed",
			Capacity:     0,
		})
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if repo.lastIn.EventName != "" || repo.lastIn.EventDescription != "" {
			t.Fatalf("event text = %q / %q, want empty",
				repo.lastIn.EventName, repo.lastIn.EventDescription)
		}
	})

	t.Run("clears omitted event text on normal", func(t *testing.T) {
		repo := &stubScheduleRepo{inserted: true}
		uc := NewSetScheduleUseCase(repo, testStoreCalendar())

		_, err := uc.Execute(context.Background(), SetScheduleCommand{
			Date:         datetime.MustParseDate("2026-05-20"),
			ScheduleType: "normal",
			Capacity:     10,
		})
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if repo.lastIn.EventName != "" || repo.lastIn.EventDescription != "" {
			t.Fatalf("event text = %q / %q, want empty",
				repo.lastIn.EventName, repo.lastIn.EventDescription)
		}
	})
}

func TestSetScheduleUseCase_Execute_externalEvent(t *testing.T) {
	t.Run("clears business hours for external_event", func(t *testing.T) {
		repo := &stubScheduleRepo{inserted: true}
		uc := NewSetScheduleUseCase(repo, testStoreCalendar())

		_, err := uc.Execute(context.Background(), SetScheduleCommand{
			Date:          datetime.MustParseDate("2026-02-11"),
			ScheduleType:  domain.ScheduleTypeExternalEvent,
			Capacity:      0,
			EventName:     "和紅茶をしばく会",
			OpenTime:      datetime.MustParseTime("11:30"),
			LastOrderTime: datetime.MustParseTime("13:30"),
			CloseTime:     datetime.MustParseTime("15:00"),
		})
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if !repo.lastIn.OpenTime.IsZero() || !repo.lastIn.LastOrderTime.IsZero() || !repo.lastIn.CloseTime.IsZero() {
			t.Fatalf("times = %s/%s/%s, want all zero",
				repo.lastIn.OpenTime, repo.lastIn.LastOrderTime, repo.lastIn.CloseTime)
		}
	})

	t.Run("forces capacity zero for external_event", func(t *testing.T) {
		repo := &stubScheduleRepo{inserted: true}
		uc := NewSetScheduleUseCase(repo, testStoreCalendar())

		_, err := uc.Execute(context.Background(), SetScheduleCommand{
			Date:         datetime.MustParseDate("2026-02-11"),
			ScheduleType: domain.ScheduleTypeExternalEvent,
			Capacity:     10,
			EventName:    "和紅茶をしばく会",
		})
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if repo.lastIn.Capacity != 0 {
			t.Fatalf("Capacity = %d, want 0", repo.lastIn.Capacity)
		}
	})

	t.Run("persists event name and optional description for external_event", func(t *testing.T) {
		repo := &stubScheduleRepo{inserted: true}
		uc := NewSetScheduleUseCase(repo, testStoreCalendar())

		_, err := uc.Execute(context.Background(), SetScheduleCommand{
			Date:         datetime.MustParseDate("2026-02-11"),
			ScheduleType: domain.ScheduleTypeExternalEvent,
			Capacity:     0,
			EventName:    "和紅茶をしばく会",
		})
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if repo.lastIn.EventName != "和紅茶をしばく会" {
			t.Fatalf("EventName = %q", repo.lastIn.EventName)
		}
		if repo.lastIn.EventDescription != "" {
			t.Fatalf("EventDescription = %q, want empty", repo.lastIn.EventDescription)
		}
	})
}
