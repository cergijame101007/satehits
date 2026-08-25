package application

import (
	"context"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// VisitDateLocker は同一来店日の予約作成を直列化するためのロック
// 実装はトランザクション内（ctx に *sql.Tx があること）でのみ有効
type VisitDateLocker interface {
	Lock(ctx context.Context, date datetime.Date) error
}
