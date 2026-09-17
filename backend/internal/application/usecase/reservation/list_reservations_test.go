package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// listTestReservationRepo は List の呼び出しを記録する（他のメソッドは fakeReservationRepo のまま）
type listTestReservationRepo struct {
	fakeReservationRepo
	listed     []domain.Reservation
	listErr    error
	listCalls  int
	lastFilter domain.ListReservationsFilter
}

func (r *listTestReservationRepo) List(_ context.Context, f domain.ListReservationsFilter) ([]domain.Reservation, error) {
	r.listCalls++
	r.lastFilter = f
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listed, nil
}

func datePtr(s string) *datetime.Date {
	d := datetime.MustParseDate(s)
	return &d
}

func TestListReservationsUseCase_Execute_filters(t *testing.T) {
	tests := []struct {
		name  string
		query ListReservationsQuery
	}{
		{name: "passes empty filter when no query is given", query: ListReservationsQuery{}},
		{name: "passes date filter to repository", query: ListReservationsQuery{Date: datePtr("2026-05-20")}},
		{name: "passes status filter to repository", query: ListReservationsQuery{Status: "pending"}},
		{name: "passes source filter to repository", query: ListReservationsQuery{Source: "instagram"}},
		{
			name:  "passes all filters to repository",
			query: ListReservationsQuery{Date: datePtr("2026-05-20"), Status: "approved", Source: "walk_in"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &listTestReservationRepo{}
			uc := NewListReservationsUseCase(repo)

			if _, err := uc.Execute(context.Background(), tt.query); err != nil {
				t.Fatalf("Execute() err = %v, want nil", err)
			}
			if repo.listCalls != 1 {
				t.Fatalf("List calls = %d, want 1", repo.listCalls)
			}
			got := repo.lastFilter
			if (got.Date == nil) != (tt.query.Date == nil) {
				t.Fatalf("filter.Date = %v, want %v", got.Date, tt.query.Date)
			}
			if got.Date != nil && *got.Date != *tt.query.Date {
				t.Fatalf("filter.Date = %s, want %s", got.Date, tt.query.Date)
			}
			if got.Status != tt.query.Status {
				t.Fatalf("filter.Status = %q, want %q", got.Status, tt.query.Status)
			}
			if got.Source != tt.query.Source {
				t.Fatalf("filter.Source = %q, want %q", got.Source, tt.query.Source)
			}
		})
	}
}

func TestListReservationsUseCase_Execute_results(t *testing.T) {
	t.Run("returns reservations of the date with total", func(t *testing.T) {
		d := datetime.MustParseDate("2026-05-20")
		repo := &listTestReservationRepo{
			listed: []domain.Reservation{
				{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), VisitDate: d, Status: "pending"},
				{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"), VisitDate: d, Status: "approved"},
			},
		}
		uc := NewListReservationsUseCase(repo)

		result, err := uc.Execute(context.Background(), ListReservationsQuery{Date: &d})
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if len(result.Reservations) != 2 {
			t.Fatalf("len(Reservations) = %d, want 2", len(result.Reservations))
		}
		if result.Total != 2 {
			t.Fatalf("Total = %d, want 2", result.Total)
		}
		if result.Reservations[0].ID != repo.listed[0].ID || result.Reservations[1].ID != repo.listed[1].ID {
			t.Fatalf("Reservations = %+v, want repository order", result.Reservations)
		}
	})

	t.Run("returns empty non-nil list when repository has no reservations", func(t *testing.T) {
		repo := &listTestReservationRepo{listed: nil}
		uc := NewListReservationsUseCase(repo)

		result, err := uc.Execute(context.Background(), ListReservationsQuery{Date: datePtr("2026-05-20")})
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if result.Reservations == nil {
			t.Fatal("Reservations = nil, want empty slice (JSON [])")
		}
		if len(result.Reservations) != 0 || result.Total != 0 {
			t.Fatalf("Reservations = %v, Total = %d, want empty and 0", result.Reservations, result.Total)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &listTestReservationRepo{listErr: repoErr}
		uc := NewListReservationsUseCase(repo)

		result, err := uc.Execute(context.Background(), ListReservationsQuery{})
		if !errors.Is(err, repoErr) {
			t.Fatalf("Execute() err = %v, want %v", err, repoErr)
		}
		if result != nil {
			t.Fatalf("result = %+v, want nil", result)
		}
	})
}

func TestListReservationsUseCase_Execute_validation(t *testing.T) {
	tests := []struct {
		name       string
		query      ListReservationsQuery
		wantFields []string
	}{
		{name: "rejects unknown status", query: ListReservationsQuery{Status: "done"}, wantFields: []string{"status"}},
		{name: "rejects unknown source", query: ListReservationsQuery{Source: "line"}, wantFields: []string{"source"}},
		{
			name:       "reports both status and source violations",
			query:      ListReservationsQuery{Status: "done", Source: "line"},
			wantFields: []string{"status", "source"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &listTestReservationRepo{}
			uc := NewListReservationsUseCase(repo)

			_, err := uc.Execute(context.Background(), tt.query)
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("Execute() err = %v, want ValidationError", err)
			}
			if len(vErr.Violations) != len(tt.wantFields) {
				t.Fatalf("violations = %+v, want fields %v", vErr.Violations, tt.wantFields)
			}
			for _, field := range tt.wantFields {
				assertHasViolationField(t, vErr.Violations, field)
			}
			if repo.listCalls != 0 {
				t.Fatalf("List calls = %d, want 0", repo.listCalls)
			}
		})
	}
}
