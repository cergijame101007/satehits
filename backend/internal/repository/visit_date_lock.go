package repository

import (
	"context"
	"fmt"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// visitDateLockNamespace は pg_advisory_xact_lock の key1。予約・来店日用
const visitDateLockNamespace int32 = 1

// PostgresVisitDateLocker は visit_date をキーにトランザクション級 advisory lock を取る
// DB は持たない。TxManager が ctx に載せた *sql.Tx だけを使う
type PostgresVisitDateLocker struct{}

// Lock は同一 visit_date の予約作成を直列化する。ctx に tx が無ければエラー
func (PostgresVisitDateLocker) Lock(ctx context.Context, date datetime.Date) error {
	tx, ok := getTx(ctx)
	if !ok {
		return fmt.Errorf("visit date lock requires a transaction")
	}
	return execVisitDateLock(ctx, tx, date)
}

func visitDateAdvisoryKeys(d datetime.Date) (namespace, yyyymmdd int32) {
	u := d.UTC()
	// YYYYMMDD は年 2147 まで int32 に収まる（予約日付の実用範囲）
	key := int32(u.Year())*10000 + int32(u.Month())*100 + int32(u.Day()) //nolint:gosec // G115: date components fit int32
	return visitDateLockNamespace, key
}

func execVisitDateLock(ctx context.Context, db DBTX, date datetime.Date) error {
	ns, key := visitDateAdvisoryKeys(date)
	_, err := db.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, ns, key)
	if err != nil {
		return fmt.Errorf("pg_advisory_xact_lock: %w", err)
	}
	return nil
}
