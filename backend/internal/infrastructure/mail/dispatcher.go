package mail

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

const (
	maxAttempts          = 6
	defaultBatchSize     = 20
	defaultSendTimeout   = 30 * time.Second
	defaultLeaseDuration = 5 * time.Minute
	// 認証エラーで中断した行は設定修正後すぐ拾えるよう短い遅延にする
	authErrorRetryDelay = 1 * time.Minute
	// mark 系は呼び出し元の cancel に巻き込まれないよう独立した短いタイムアウトで行う
	markTimeout = 10 * time.Second
)

// Halted の理由
const (
	HaltAuthError  = "auth_error"
	HaltTimeBudget = "time_budget"
)

// attempt_count は「claim した回数（今回の試行を含む）」。
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
	// Errors は送信結果の記録（mark）に失敗した件数。行は lease 中のまま残り、lease 切れ後に再 claim される
	Errors int `json:"errors"`
	// Halted はバッチを途中で打ち切った理由（auth_error / time_budget）。打ち切りが無ければ空
	Halted string `json:"halted,omitempty"`
}

// Config は Dispatcher の動作パラメータ。ゼロ値は既定値に置き換える
type Config struct {
	// BatchSize は 1 回の ProcessPending で claim する最大件数
	BatchSize int
	// TimeBudget は 1 回の ProcessPending に使える時間。残りが SendTimeout 未満なら次の claim をしない。0 は無制限
	TimeBudget time.Duration
	// SendTimeout は 1 通あたりの送信タイムアウト
	SendTimeout time.Duration
	// LeaseDuration は claim から再 claim 可能になるまでの時間。SendTimeout より十分長くする
	LeaseDuration time.Duration
}

// Dispatcher は Outbox の pending 行を送信する（goroutine / ticker なし）。
// claim → 送信 → mark をそれぞれ独立してコミットする lease 方式で、
// 送信中のクラッシュや cancel でも試行回数は消費済み・lease 切れ後に再試行される
type Dispatcher struct {
	repo          domain.EmailOutboxRepository
	sender        domain.MailSender
	batchSize     int
	timeBudget    time.Duration
	sendTimeout   time.Duration
	leaseDuration time.Duration
	now           func() time.Time
}

// NewDispatcher は Dispatcher を生成する
func NewDispatcher(repo domain.EmailOutboxRepository, sender domain.MailSender, cfg Config) *Dispatcher {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultBatchSize
	}
	if cfg.SendTimeout <= 0 {
		cfg.SendTimeout = defaultSendTimeout
	}
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = defaultLeaseDuration
	}
	if cfg.LeaseDuration <= cfg.SendTimeout {
		// 送信中に lease が切れると同じ行を二重に claim しうるため、必ず送信タイムアウトより長くする
		cfg.LeaseDuration = cfg.SendTimeout + defaultLeaseDuration
	}
	return &Dispatcher{
		repo:          repo,
		sender:        sender,
		batchSize:     cfg.BatchSize,
		timeBudget:    cfg.TimeBudget,
		sendTimeout:   cfg.SendTimeout,
		leaseDuration: cfg.LeaseDuration,
		now:           time.Now,
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

// ProcessPending は送信可能な行を最大 BatchSize 件・TimeBudget 内で処理する。
// claim 自体の失敗（DB 断など）だけをエラーとして返し、行ごとの mark 失敗は Errors に数えて続行する
func (d *Dispatcher) ProcessPending(ctx context.Context) (ProcessStats, error) {
	var stats ProcessStats
	start := d.now()
	for stats.Processed < d.batchSize {
		if d.overBudget(start) {
			stats.Halted = HaltTimeBudget
			log.Printf("mail outbox halted: reason=%s processed=%d", HaltTimeBudget, stats.Processed)
			break
		}
		msg, err := d.repo.ClaimNextPending(ctx, d.now().Add(d.leaseDuration))
		if err != nil {
			return stats, fmt.Errorf("process outbox: %w", err)
		}
		if msg == nil {
			break
		}
		stats.Processed++
		if halt := d.deliver(ctx, msg, &stats); halt != "" {
			stats.Halted = halt
			break
		}
	}
	return stats, nil
}

func (d *Dispatcher) overBudget(start time.Time) bool {
	if d.timeBudget <= 0 {
		return false
	}
	return d.now().Sub(start)+d.sendTimeout > d.timeBudget
}

// deliver は claim 済みの 1 行を送信し結果を記録する。バッチを中断すべきときは理由を返す
func (d *Dispatcher) deliver(ctx context.Context, msg *domain.EmailOutboxMessage, stats *ProcessStats) string {
	sendCtx, cancelSend := context.WithTimeout(ctx, d.sendTimeout)
	sendErr := d.sender.Send(sendCtx, domain.MailMessage{
		From:           msg.FromAddress,
		To:             msg.ToAddress,
		Subject:        msg.Subject,
		HTML:           msg.BodyHTML,
		Text:           msg.BodyText,
		IdempotencyKey: domain.IdempotencyKey(msg.MailType, msg.ReservationID),
	})
	cancelSend()

	// 送信は済んでいる可能性があるため、記録は呼び出し元の cancel から切り離す
	markCtx, cancelMark := context.WithTimeout(context.WithoutCancel(ctx), markTimeout)
	defer cancelMark()

	if sendErr == nil {
		if err := d.repo.MarkSent(markCtx, msg.ID); err != nil {
			d.recordMarkError(stats, msg, "mark sent", err)
			return ""
		}
		stats.Sent++
		return ""
	}

	errText := sendErr.Error()
	if errors.Is(sendErr, domain.ErrMailAuth) {
		if err := d.repo.ReleaseClaim(markCtx, msg.ID, d.now().Add(authErrorRetryDelay), errText); err != nil {
			d.recordMarkError(stats, msg, "release claim", err)
		}
		log.Printf("ERROR: mail outbox halted: reason=%s id=%s type=%s reservation=%s err=%v",
			HaltAuthError, msg.ID, msg.MailType, msg.ReservationID, sendErr)
		return HaltAuthError
	}

	if errors.Is(sendErr, domain.ErrMailPermanent) || ShouldMarkFailed(msg.AttemptCount) {
		d.markFailed(markCtx, msg, stats, sendErr)
		return ""
	}

	nextAt, ok := NextRetryAt(msg.AttemptCount, d.now())
	if !ok {
		d.markFailed(markCtx, msg, stats, sendErr)
		return ""
	}
	if err := d.repo.MarkRetry(markCtx, msg.ID, nextAt, errText); err != nil {
		d.recordMarkError(stats, msg, "mark retry", err)
		return ""
	}
	stats.Retried++
	log.Printf("mail outbox retry: id=%s type=%s reservation=%s attempt=%d next=%s err=%v",
		msg.ID, msg.MailType, msg.ReservationID, msg.AttemptCount, nextAt.Format(time.RFC3339), sendErr)
	return ""
}

func (d *Dispatcher) markFailed(ctx context.Context, msg *domain.EmailOutboxMessage, stats *ProcessStats, sendErr error) {
	if err := d.repo.MarkFailed(ctx, msg.ID, sendErr.Error()); err != nil {
		d.recordMarkError(stats, msg, "mark failed", err)
		return
	}
	stats.Failed++
	log.Printf("ERROR: mail outbox failed: id=%s type=%s reservation=%s attempt=%d err=%v",
		msg.ID, msg.MailType, msg.ReservationID, msg.AttemptCount, sendErr)
}

func (d *Dispatcher) recordMarkError(stats *ProcessStats, msg *domain.EmailOutboxMessage, op string, err error) {
	stats.Errors++
	log.Printf("ERROR: mail outbox %s: id=%s type=%s reservation=%s attempt=%d err=%v",
		op, msg.ID, msg.MailType, msg.ReservationID, msg.AttemptCount, err)
}
