package domain

import (
	"context"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// PendingReminderReservationReader はオーナー向け pending リマインド（UC-S04）が必要とする予約の読み取り。
// ReservationRepository とは分離し、リマインド専用の絞り込みを既存の実装・fake に影響させない
type PendingReminderReservationReader interface {
	// ListPendingWebByVisitDateRange は status=pending かつ source=web で来店日が [from, to] の予約を返す
	ListPendingWebByVisitDateRange(ctx context.Context, from, to datetime.Date) ([]Reservation, error)
	// CountPendingFrom は来店日が from 以降の pending 予約数（source を問わない）を返す
	CountPendingFrom(ctx context.Context, from datetime.Date) (int, error)
}
