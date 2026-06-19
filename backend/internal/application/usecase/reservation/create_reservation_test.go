package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/application"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

type passThroughTxManager struct{}

func (passThroughTxManager) DoInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

var _ application.TxManager = passThroughTxManager{}

type createTestScheduleRepo struct {
	byDate map[string]domain.Schedule
}

func (r createTestScheduleRepo) Upsert(context.Context, domain.SetScheduleInput) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (r createTestScheduleRepo) FindByDate(_ context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	s, ok := r.byDate[date.String()]
	return s, ok, nil
}

func (r createTestScheduleRepo) ListStoredByYearMonth(context.Context, int, int) ([]domain.Schedule, error) {
	return nil, nil
}

type createTestReservationRepo struct {
	approvedByDate map[string]int
	created        []domain.CreateReservationInput
}

func (r *createTestReservationRepo) Create(_ context.Context, in domain.CreateReservationInput) (domain.Reservation, error) {
	r.created = append(r.created, in)
	return domain.Reservation{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:      in.Name,
		People:    in.People,
		VisitDate: in.VisitDate,
		VisitTime: in.VisitTime,
		Status:    in.Status,
		Source:    in.Source,
	}, nil
}

func (r *createTestReservationRepo) GetAll(context.Context) ([]domain.Reservation, error) {
	return nil, nil
}

func (r *createTestReservationRepo) List(context.Context, domain.ListReservationsFilter) ([]domain.Reservation, error) {
	return nil, nil
}

func (r *createTestReservationRepo) GetByID(context.Context, uuid.UUID) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r *createTestReservationRepo) UpdateStatus(context.Context, uuid.UUID, string) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r *createTestReservationRepo) SumReservedPeopleByDate(_ context.Context, date datetime.Date) (int, error) {
	if r.approvedByDate == nil {
		return 0, nil
	}
	return r.approvedByDate[date.String()], nil
}

func (r *createTestReservationRepo) SumReservedPeopleByDateRange(_ context.Context, from, to datetime.Date) (map[string]int, error) {
	if r.approvedByDate == nil {
		return map[string]int{}, nil
	}
	result := make(map[string]int)
	fromStr := from.String()
	toStr := to.String()
	for dateStr, count := range r.approvedByDate {
		if dateStr >= fromStr && dateStr <= toStr {
			result[dateStr] = count
		}
	}
	return result, nil
}

func newCreateReservationUseCaseForTest(sched createTestScheduleRepo, repo *createTestReservationRepo) *CreateReservationUseCase {
	resolver := service.NewScheduleResolver(sched)
	avail := service.NewAvailabilityService(resolver, repo)
	return NewCreateReservationUseCase(repo, resolver, avail, passThroughTxManager{}, NoOpCaptchaVerifier{}, NoOpMailNotifier{})
}

// firstBookableWeekday は JST 基準で翌日〜14日先の範囲内で最初の指定曜日を返す
func firstBookableWeekday(t *testing.T, now time.Time, wd time.Weekday) datetime.Date {
	t.Helper()
	nowDay := now.In(storeLocation)
	today := time.Date(nowDay.Year(), nowDay.Month(), nowDay.Day(), 0, 0, 0, 0, storeLocation)
	for lead := minBookingLeadDays; lead <= maxBookingLeadDays; lead++ {
		candidate := today.AddDate(0, 0, lead)
		if candidate.Weekday() == wd {
			return datetime.NewDate(candidate.Year(), candidate.Month(), candidate.Day())
		}
	}
	t.Fatalf("no bookable %v within lead days %d..%d from %s", wd, minBookingLeadDays, maxBookingLeadDays, today.Format("2006-01-02"))
	return datetime.Date{}
}

func visitTimeForWeekday(wd time.Weekday) datetime.Time {
	switch wd {
	case time.Saturday, time.Sunday:
		return datetime.MustParseTime("12:00")
	default:
		return datetime.MustParseTime("12:00")
	}
}

func validCreateCommandForDate(date datetime.Date) CreateReservationCommand {
	cmd := validCreateReservationCommand()
	cmd.VisitDate = date
	cmd.VisitTime = visitTimeForWeekday(civilWeekday(date))
	return cmd
}

func TestCreateReservationUseCase_Execute_availability(t *testing.T) {
	now := time.Now().In(storeLocation)
	sunday := firstBookableWeekday(t, now, time.Sunday)

	t.Run("creates reservation when people equals available", func(t *testing.T) {
		repo := &createTestReservationRepo{approvedByDate: map[string]int{sunday.String(): 8}}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {Date: sunday, ScheduleType: domain.ScheduleTypeMorning, Capacity: 10},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)
		cmd.People = 2

		result, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if result == nil {
			t.Fatal("result = nil, want reservation")
		}
		if len(repo.created) != 1 {
			t.Fatalf("created count = %d, want 1", len(repo.created))
		}
		if repo.created[0].People != 2 {
			t.Fatalf("created.People = %d, want 2", repo.created[0].People)
		}
	})

	t.Run("creates reservation when people is below available", func(t *testing.T) {
		repo := &createTestReservationRepo{approvedByDate: map[string]int{sunday.String(): 3}}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {Date: sunday, ScheduleType: domain.ScheduleTypeMorning, Capacity: 10},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)
		cmd.People = 1

		_, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
	})

	t.Run("returns capacity exceeded when people exceeds available by one", func(t *testing.T) {
		repo := &createTestReservationRepo{approvedByDate: map[string]int{sunday.String(): 8}}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {Date: sunday, ScheduleType: domain.ScheduleTypeMorning, Capacity: 10},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)
		cmd.People = 3

		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("Execute() err = nil, want ErrCapacityExceeded")
		}
		if !errors.Is(err, domain.ErrCapacityExceeded) {
			t.Fatalf("Execute() err = %v, want ErrCapacityExceeded", err)
		}
		if len(repo.created) != 0 {
			t.Fatalf("created count = %d, want 0", len(repo.created))
		}
	})

	t.Run("returns capacity exceeded when available is zero", func(t *testing.T) {
		repo := &createTestReservationRepo{approvedByDate: map[string]int{sunday.String(): 10}}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {Date: sunday, ScheduleType: domain.ScheduleTypeMorning, Capacity: 10},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)
		cmd.People = 1

		_, err := uc.Execute(context.Background(), cmd)
		if !errors.Is(err, domain.ErrCapacityExceeded) {
			t.Fatalf("Execute() err = %v, want ErrCapacityExceeded", err)
		}
	})

	t.Run("accepts max people when exactly enough seats remain", func(t *testing.T) {
		repo := &createTestReservationRepo{approvedByDate: map[string]int{sunday.String(): 3}}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {Date: sunday, ScheduleType: domain.ScheduleTypeMorning, Capacity: 10},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)
		cmd.People = 7

		_, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
	})

	t.Run("rejects closed day from daily_schedules", func(t *testing.T) {
		repo := &createTestReservationRepo{}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {Date: sunday, ScheduleType: domain.ScheduleTypeClosed, Capacity: 0},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)

		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("Execute() err = nil, want ValidationError")
		}
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("Execute() err = %v, want ValidationError", err)
		}
		assertHasViolationField(t, vErr.Violations, "visit_date")
		if len(repo.created) != 0 {
			t.Fatalf("created count = %d, want 0", len(repo.created))
		}
	})

	t.Run("rejects external_event day with visit_date", func(t *testing.T) {
		repo := &createTestReservationRepo{}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {
					Date:         sunday,
					ScheduleType: domain.ScheduleTypeExternalEvent,
					Capacity:     0,
					EventName:    "和紅茶をしばく会",
				},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)
		cmd.People = 1

		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("Execute() err = nil, want ValidationError")
		}
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("Execute() err = %v, want ValidationError", err)
		}
		assertHasViolationField(t, vErr.Violations, "visit_date")
	})

	t.Run("accepts in_store event day with available capacity", func(t *testing.T) {
		repo := &createTestReservationRepo{}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {
					Date:         sunday,
					ScheduleType: domain.ScheduleTypeEvent,
					Capacity:     10,
					EventName:    "店内イベント",
				},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)
		cmd.People = 2

		_, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if len(repo.created) != 1 {
			t.Fatalf("created count = %d, want 1", len(repo.created))
		}
	})

	t.Run("accepts Thursday when daily_schedules overrides to normal", func(t *testing.T) {
		thursday := firstBookableWeekday(t, now, time.Thursday)
		repo := &createTestReservationRepo{}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				thursday.String(): {Date: thursday, ScheduleType: domain.ScheduleTypeNormal, Capacity: 10},
			},
		}, repo)

		cmd := validCreateCommandForDate(thursday)
		cmd.VisitTime = datetime.MustParseTime("12:00")

		_, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if len(repo.created) != 1 {
			t.Fatalf("created count = %d, want 1", len(repo.created))
		}
	})

	t.Run("rejects default closed Thursday with visit_date", func(t *testing.T) {
		thursday := firstBookableWeekday(t, now, time.Thursday)
		repo := &createTestReservationRepo{}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{}, repo)

		cmd := validCreateCommandForDate(thursday)
		cmd.VisitTime = datetime.MustParseTime("12:00")

		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("Execute() err = nil, want ValidationError")
		}
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("Execute() err = %v, want ValidationError", err)
		}
		assertHasViolationField(t, vErr.Violations, "visit_date")
	})

	t.Run("rejects Sunday morning coffee time before lunch open", func(t *testing.T) {
		repo := &createTestReservationRepo{}
		uc := newCreateReservationUseCaseForTest(createTestScheduleRepo{
			byDate: map[string]domain.Schedule{
				sunday.String(): {Date: sunday, ScheduleType: domain.ScheduleTypeMorning, Capacity: 10},
			},
		}, repo)

		cmd := validCreateCommandForDate(sunday)
		cmd.VisitTime = datetime.MustParseTime("09:00")

		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("Execute() err = nil, want ValidationError")
		}
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("Execute() err = %v, want ValidationError", err)
		}
		assertHasViolationField(t, vErr.Violations, "visit_time")
		if len(repo.created) != 0 {
			t.Fatalf("created count = %d, want 0", len(repo.created))
		}
	})
}

type fakeCaptchaVerifier struct {
	err error
}

func (f fakeCaptchaVerifier) Verify(context.Context, string) error {
	return f.err
}

func TestCreateReservationUseCase_Execute_captcha(t *testing.T) {
	now := time.Now().In(storeLocation)
	sunday := firstBookableWeekday(t, now, time.Sunday)
	repo := &createTestReservationRepo{approvedByDate: map[string]int{sunday.String(): 3}}
	sched := createTestScheduleRepo{
		byDate: map[string]domain.Schedule{
			sunday.String(): {Date: sunday, ScheduleType: domain.ScheduleTypeMorning, Capacity: 10},
		},
	}
	resolver := service.NewScheduleResolver(sched)
	avail := service.NewAvailabilityService(resolver, repo)
	cmd := validCreateCommandForDate(sunday)

	t.Run("creates reservation when captcha verification succeeds", func(t *testing.T) {
		uc := NewCreateReservationUseCase(repo, resolver, avail, passThroughTxManager{}, fakeCaptchaVerifier{}, NoOpMailNotifier{})
		_, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
	})

	t.Run("returns captcha failed when verification fails", func(t *testing.T) {
		uc := NewCreateReservationUseCase(repo, resolver, avail, passThroughTxManager{}, fakeCaptchaVerifier{err: domain.ErrCaptchaFailed}, NoOpMailNotifier{})
		_, err := uc.Execute(context.Background(), cmd)
		if !errors.Is(err, domain.ErrCaptchaFailed) {
			t.Fatalf("Execute() err = %v, want ErrCaptchaFailed", err)
		}
	})
}
