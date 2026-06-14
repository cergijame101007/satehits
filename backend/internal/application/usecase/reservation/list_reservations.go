package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// ListReservationsQuery は管理者予約一覧のクエリ
type ListReservationsQuery struct {
	Date   *datetime.Date
	Status string
	Source string
}

// ListReservationsResult は管理者予約一覧の結果
type ListReservationsResult struct {
	Reservations []domain.Reservation
	Total        int
}

// ListReservationsUseCase は管理者向け予約一覧取得
type ListReservationsUseCase struct {
	repo domain.ReservationRepository
}

// NewListReservationsUseCase は ListReservationsUseCase を生成する
func NewListReservationsUseCase(repo domain.ReservationRepository) *ListReservationsUseCase {
	return &ListReservationsUseCase{repo: repo}
}

// Execute は絞り込み条件を検証し予約一覧を返す
func (u *ListReservationsUseCase) Execute(ctx context.Context, q ListReservationsQuery) (*ListReservationsResult, error) {
	var violations []FieldViolation
	violations = append(violations, validateReservationStatus(q.Status)...)
	violations = append(violations, validateReservationSourceOptional(q.Source)...)
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	reservations, err := u.repo.List(ctx, domain.ListReservationsFilter{
		Date:   q.Date,
		Status: q.Status,
		Source: q.Source,
	})
	if err != nil {
		return nil, err
	}
	if reservations == nil {
		reservations = []domain.Reservation{}
	}

	return &ListReservationsResult{
		Reservations: reservations,
		Total:        len(reservations),
	}, nil
}

// validateReservationSourceOptional は source クエリが空ならスキップ、値ありなら enum 検証
func validateReservationSourceOptional(value string) []FieldViolation {
	if value == "" {
		return nil
	}
	if !isValidEnum(value, domain.ValidReservationSources) {
		return []FieldViolation{{Field: "source", Message: "予約経路の値が不正です"}}
	}
	return nil
}
