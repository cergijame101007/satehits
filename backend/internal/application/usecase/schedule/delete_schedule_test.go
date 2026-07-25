package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

type deleteScheduleStubRepo struct {
	byDate       map[string]domain.Schedule
	deletedDate  string
	deleteCalled bool
}

func (s *deleteScheduleStubRepo) Upsert(context.Context, domain.SetScheduleInput) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (s *deleteScheduleStubRepo) FindByDate(_ context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	sch, ok := s.byDate[date.String()]
	return sch, ok, nil
}

func (s *deleteScheduleStubRepo) ListStoredByYearMonth(context.Context, int, int) ([]domain.Schedule, error) {
	return nil, nil
}

func (s *deleteScheduleStubRepo) DeleteByDate(_ context.Context, date datetime.Date) error {
	s.deleteCalled = true
	s.deletedDate = date.String()
	if _, ok := s.byDate[date.String()]; !ok {
		return domain.ErrScheduleNotStored
	}
	delete(s.byDate, date.String())
	return nil
}

func TestDeleteScheduleUseCase_Execute_deletesStoredRow(t *testing.T) {
	d := datetime.MustParseDate("2026-05-18")
	repo := &deleteScheduleStubRepo{
		byDate: map[string]domain.Schedule{
			d.String(): {Date: d, ScheduleType: domain.ScheduleTypeEvent, Capacity: 10},
		},
	}
	uc := NewDeleteScheduleUseCase(repo)

	if err := uc.Execute(context.Background(), d); err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if !repo.deleteCalled {
		t.Fatal("DeleteByDate was not called")
	}
	if repo.deletedDate != d.String() {
		t.Fatalf("deletedDate = %q, want %q", repo.deletedDate, d.String())
	}
	if _, ok := repo.byDate[d.String()]; ok {
		t.Fatal("row still present after delete")
	}
}

func TestDeleteScheduleUseCase_Execute_returnsNotStoredWhenMissing(t *testing.T) {
	d := datetime.MustParseDate("2026-05-18")
	repo := &deleteScheduleStubRepo{byDate: map[string]domain.Schedule{}}
	uc := NewDeleteScheduleUseCase(repo)

	err := uc.Execute(context.Background(), d)
	if !errors.Is(err, domain.ErrScheduleNotStored) {
		t.Fatalf("err = %v, want ErrScheduleNotStored", err)
	}
	if repo.deleteCalled {
		t.Fatal("DeleteByDate should not be called when row is missing")
	}
}

func TestDeleteScheduleUseCase_Execute_rejectsZeroDate(t *testing.T) {
	uc := NewDeleteScheduleUseCase(&deleteScheduleStubRepo{byDate: map[string]domain.Schedule{}})

	err := uc.Execute(context.Background(), datetime.Date{})
	if err == nil {
		t.Fatal("err = nil, want ValidationError")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("err type = %T, want *ValidationError", err)
	}
	if len(vErr.Violations) == 0 || vErr.Violations[0].Field != "date" {
		t.Fatalf("violations = %+v, want date field", vErr.Violations)
	}
}
