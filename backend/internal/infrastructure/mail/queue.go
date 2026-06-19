package mail

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

var mailSendTimeout = 30 * time.Second

// Queue はインプロセス非同期メール送信キュー
type Queue struct {
	ch     chan domain.MailMessage
	sender domain.MailSender
	wg     sync.WaitGroup
	once   sync.Once
}

// NewQueue はバッファ付きキューを生成する
func NewQueue(sender domain.MailSender, bufferSize int) *Queue {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &Queue{
		ch:     make(chan domain.MailMessage, bufferSize),
		sender: sender,
	}
}

// Start はワーカー goroutine を起動する
func (q *Queue) Start() {
	q.wg.Add(1)
	go q.run()
}

func (q *Queue) run() {
	defer q.wg.Done()
	for msg := range q.ch {
		ctx, cancel := context.WithTimeout(context.Background(), mailSendTimeout)
		err := q.sender.Send(ctx, msg)
		cancel()
		if err != nil {
			log.Printf("mail send failed: to=%s subject=%q err=%v", msg.To, msg.Subject, err)
		}
	}
}

// Enqueue はメールを非同期送信キューへ投入する（満杯時はログしてドロップ）
func (q *Queue) Enqueue(msg domain.MailMessage) {
	select {
	case q.ch <- msg:
	default:
		log.Printf("mail queue full: dropped to=%s subject=%q", msg.To, msg.Subject)
	}
}

// Shutdown はキューを閉じ、残件の送信完了を待つ
func (q *Queue) Shutdown(ctx context.Context) {
	q.once.Do(func() {
		close(q.ch)
	})
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		log.Printf("mail queue shutdown timed out: %v", ctx.Err())
	}
}
