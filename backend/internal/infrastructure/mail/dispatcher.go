package mail

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cergijame101007/satehits/internal/application"
	"github.com/cergijame101007/satehits/internal/domain"
)

const maxAttempts = 6

// attempt_count は「実際に送信を試行して失敗した回数」。
// 失敗直後の attempt_count（1〜5）に対応する次回待機時間。
var retryBackoff = []time.Duration{
	1 * time.Minute,  // attempt_count が 1 になった直後 → +1m
	5 * time.Minute,  // 2 → +5m
	15 * time.Minute, // 3 → +15m
	1 * time.Hour,    // 4 → +1h
	4 * time.Hour,    // 5 → +4h
}

// ProcessStats は 1 回の ProcessPending の集計
type ProcessStats struct {
	Processed int `json:"processed"`
	Sent      int `json:"sent"`
	Retried   int `json:"retried"`
	Failed    int `json:"failed"`
}

// Dispatcher は Outbox の pending 行を送信する（goroutine / ticker なし）
type Dispatcher struct {
	repo      domain.EmailOutboxRepository
	sender    domain.MailSender
	txManager application.TxManager
	batchSize int
	now       func() time.Time
}

// NewDispatcher は Dispatcher を生成する
func NewDispatcher(repo domain.EmailOutboxRepository, sender domain.MailSender, txManager application.TxManager, batchSize int) *Dispatcher {
	if batchSize <= 0 {
		batchSize = 20
	}
	return &Dispatcher{
		repo:      repo,
		sender:    sender,
		txManager: txManager,
		batchSize: batchSize,
		now:       time.Now,
	}
}

// NextRetryAt は失敗直後の attempt_count（1..5）から次回試行時刻を返す。
// attempt_count >= maxAttempts のときはゼロ値（呼び出し側で MarkFailed する）。
func NextRetryAt(attemptCount int, now time.Time) (time.Time, bool) {
	if attemptCount < 1 || attemptCount >= maxAttempts {
		return time.Time{}, false
	}
	idx := attemptCount - 1
	if idx >= len(retryBackoff) {
		return time.Time{}, false
	}
	return now.Add(retryBackoff[idx]), true
}

// ShouldMarkFailed は失敗直後の attempt_count が再送上限に達したか判定する
func ShouldMarkFailed(attemptCountAfterFailure int) bool {
	return attemptCountAfterFailure >= maxAttempts
}

// ProcessPending は送信可能な行を最大 batchSize 件まで処理する
func (d *Dispatcher) ProcessPending(ctx context.Context) (ProcessStats, error) {
	var stats ProcessStats
	for stats.Processed < d.batchSize {
		done, err := d.processOne(ctx, &stats)
		if err != nil {
			return stats, err
		}
		if done {
			break
		}
	}
	return stats, nil
}

// processOne は 1 行を 1 トランザクションで claim → 送信 → mark する。
// 対象が無ければ done=true。
func (d *Dispatcher) processOne(ctx context.Context, stats *ProcessStats) (done bool, err error) {
	err = d.txManager.DoInTx(ctx, func(txCtx context.Context) error {
		msg, claimErr := d.repo.ClaimNextPending(txCtx)
		if claimErr != nil {
			return claimErr
		}
		if msg == nil {
			done = true
			return nil
		}
		stats.Processed++

		sendErr := d.sender.Send(txCtx, domain.MailMessage{
			From:           msg.FromAddress,
			To:             msg.ToAddress,
			Subject:        msg.Subject,
			HTML:           msg.BodyHTML,
			Text:           msg.BodyText,
			IdempotencyKey: domain.IdempotencyKey(msg.MailType, msg.ReservationID),
		})
		if sendErr == nil {
			if markErr := d.repo.MarkSent(txCtx, msg.ID); markErr != nil {
				return markErr
			}
			stats.Sent++
			return nil
		}

		nextAttempt := msg.AttemptCount + 1
		errText := sendErr.Error()
		if errors.Is(sendErr, domain.ErrMailPermanent) || ShouldMarkFailed(nextAttempt) {
			if markErr := d.repo.MarkFailed(txCtx, msg.ID, nextAttempt, errText); markErr != nil {
				return markErr
			}
			stats.Failed++
			log.Printf("mail outbox failed: id=%s type=%s reservation=%s attempt=%d err=%v",
				msg.ID, msg.MailType, msg.ReservationID, nextAttempt, sendErr)
			return nil
		}

		nextAt, ok := NextRetryAt(nextAttempt, d.now())
		if !ok {
			if markErr := d.repo.MarkFailed(txCtx, msg.ID, nextAttempt, errText); markErr != nil {
				return markErr
			}
			stats.Failed++
			return nil
		}
		if markErr := d.repo.MarkRetry(txCtx, msg.ID, nextAttempt, nextAt, errText); markErr != nil {
			return markErr
		}
		stats.Retried++
		log.Printf("mail outbox retry: id=%s type=%s reservation=%s attempt=%d next=%s err=%v",
			msg.ID, msg.MailType, msg.ReservationID, nextAttempt, nextAt.Format(time.RFC3339), sendErr)
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("process outbox: %w", err)
	}
	return done, nil
}
