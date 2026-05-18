package usecase

import (
	"context"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

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

func TestSetScheduleUseCase_Execute_appliesDefaultBusinessHours(t *testing.T) {
	repo := &stubScheduleRepo{inserted: true}
	uc := NewSetScheduleUseCase(repo)

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
	if repo.lastIn.LastOrderTime.String() != "14:00" {
		t.Fatalf("LastOrderTime = %s, want 14:00", repo.lastIn.LastOrderTime.String())
	}
	if repo.lastIn.CloseTime.String() != "15:00" {
		t.Fatalf("CloseTime = %s, want 15:00", repo.lastIn.CloseTime.String())
	}
}

func TestSetScheduleUseCase_Execute_appliesWeekdayDefaultsForEvent(t *testing.T) {
	repo := &stubScheduleRepo{inserted: true}
	uc := NewSetScheduleUseCase(repo)

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
	if repo.lastIn.LastOrderTime.String() != "14:00" {
		t.Fatalf("LastOrderTime = %s, want 14:00", repo.lastIn.LastOrderTime.String())
	}
	if repo.lastIn.CloseTime.String() != "15:00" {
		t.Fatalf("CloseTime = %s, want 15:00", repo.lastIn.CloseTime.String())
	}
}

func TestSetScheduleUseCase_Execute_clearsTimesForClosed(t *testing.T) {
	repo := &stubScheduleRepo{inserted: true}
	uc := NewSetScheduleUseCase(repo)

	_, err := uc.Execute(context.Background(), SetScheduleCommand{
		Date:          datetime.MustParseDate("2026-05-21"),
		ScheduleType:  "closed",
		Capacity:      0,
		OpenTime:      datetime.MustParseTime("11:30"),
		LastOrderTime: datetime.MustParseTime("14:00"),
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
