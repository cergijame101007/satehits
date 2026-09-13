package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// リマインドの対象ウィンドウ（店舗カレンダーの「今日」からの日数）。
// 3 日前・前日の 1 日だけを対象にすると Scheduler が 1 日欠けたときや来店 2 日前に申請された予約が
// 通知されないため、それぞれ 2 日幅にする。mail_type ごとの UNIQUE があるので幅を広げても二重送信にはならない
const (
	pendingReminder1DMaxLeadDays = 1 // pending_reminder_1d: today .. today+1
	pendingReminder3DMaxLeadDays = 3 // pending_reminder_3d: today+2 .. today+3
)

// PendingReminderEnqueuer はオーナー向け pending リマインド（UC-S04）を Outbox へ記録する
type PendingReminderEnqueuer interface {
	EnqueuePendingReminder(ctx context.Context, r domain.Reservation, mailType domain.MailType, totalPending int) error
}

// EnqueuePendingRemindersResult は 1 回の実行の集計
type EnqueuePendingRemindersResult struct {
	// Candidates はウィンドウ内の pending（web）予約数
	Candidates int `json:"candidates"`
	// Enqueued は今回新たに Outbox へ記録した件数
	Enqueued int `json:"enqueued"`
	// Skipped は同じ mail_type が既に記録済みで飛ばした件数（再実行・並行実行で発生する正常系）
	Skipped int `json:"skipped"`
}

// EnqueuePendingRemindersUseCase は来店が近い pending（web）予約をタイミングごとに Outbox へ記録する。
// 送信は Outbox Dispatcher に任せ、冪等性は UNIQUE (reservation_id, mail_type) に委ねるためトランザクションは使わない
type EnqueuePendingRemindersUseCase struct {
	reader   domain.PendingReminderReservationReader
	enqueuer PendingReminderEnqueuer
	now      func() time.Time
}

// NewEnqueuePendingRemindersUseCase は EnqueuePendingRemindersUseCase を生成する
func NewEnqueuePendingRemindersUseCase(
	reader domain.PendingReminderReservationReader,
	enqueuer PendingReminderEnqueuer,
) *EnqueuePendingRemindersUseCase {
	return &EnqueuePendingRemindersUseCase{reader: reader, enqueuer: enqueuer, now: time.Now}
}

// Execute はウィンドウ内の対象予約を列挙し、予約ごとにリマインドを enqueue する。
// 既に記録済み（ErrMailAlreadyEnqueued）は skipped として続行し、それ以外のエラーは中断して返す
func (u *EnqueuePendingRemindersUseCase) Execute(ctx context.Context) (EnqueuePendingRemindersResult, error) {
	var result EnqueuePendingRemindersResult
	today := storeToday(u.now())

	candidates, err := u.reader.ListPendingWebByVisitDateRange(ctx, today, today.AddDays(pendingReminder3DMaxLeadDays))
	if err != nil {
		return result, fmt.Errorf("list pending reminder candidates: %w", err)
	}
	result.Candidates = len(candidates)
	if len(candidates) == 0 {
		return result, nil
	}

	totalPending, err := u.reader.CountPendingFrom(ctx, today)
	if err != nil {
		return result, fmt.Errorf("count pending reservations: %w", err)
	}

	for _, r := range candidates {
		mailType, ok := pendingReminderMailType(today, r.VisitDate)
		if !ok {
			// reader の範囲指定と一致するはずなので、ここに来るのは reader 実装の不整合
			log.Printf("WARN: pending reminder candidate outside window: reservation_id=%s visit_date=%s today=%s", r.ID, r.VisitDate, today)
			continue
		}
		err := u.enqueuer.EnqueuePendingReminder(ctx, r, mailType, totalPending)
		if errors.Is(err, domain.ErrMailAlreadyEnqueued) {
			result.Skipped++
			continue
		}
		if err != nil {
			return result, fmt.Errorf("enqueue %s for reservation %s: %w", mailType, r.ID, err)
		}
		result.Enqueued++
	}
	return result, nil
}

// storeToday は now を店舗カレンダー（Asia/Tokyo）の暦日に正規化する
func storeToday(now time.Time) datetime.Date {
	local := now.In(storeLocation)
	return datetime.NewDate(local.Year(), local.Month(), local.Day())
}

// pendingReminderMailType は来店日までの日数からタイミング種別を決める。ウィンドウ外なら ok=false
func pendingReminderMailType(today, visitDate datetime.Date) (domain.MailType, bool) {
	leadDays := int(visitDate.UTC().Sub(today.UTC()).Hours() / 24)
	switch {
	case leadDays < 0:
		return "", false
	case leadDays <= pendingReminder1DMaxLeadDays:
		return domain.MailTypePendingReminder1D, true
	case leadDays <= pendingReminder3DMaxLeadDays:
		return domain.MailTypePendingReminder3D, true
	default:
		return "", false
	}
}
