package mail

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

type recordingSender struct {
	mu    sync.Mutex
	calls []domain.MailMessage
}

func (r *recordingSender) Send(_ context.Context, msg domain.MailMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, msg)
	return nil
}

func (r *recordingSender) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *recordingSender) last() domain.MailMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[len(r.calls)-1]
}

func TestQueueEnqueueAndSend(t *testing.T) {
	sender := &recordingSender{}
	queue := NewQueue(sender, 4)
	queue.Start()

	queue.Enqueue(domain.MailMessage{
		From:    "noreply@satehits.com",
		To:      "a@example.com",
		Subject: "test",
		Text:    "hello",
	})

	deadline := time.Now().Add(2 * time.Second)
	for sender.len() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if sender.len() != 1 {
		t.Fatalf("sender calls = %d, want 1", sender.len())
	}
	msg := sender.last()
	if msg.To != "a@example.com" || msg.Subject != "test" {
		t.Fatalf("unexpected message: %+v", msg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	queue.Shutdown(ctx)
}

func TestQueueDropWhenFull(t *testing.T) {
	block := make(chan struct{})
	sender := &blockingSender{block: block}
	queue := NewQueue(sender, 1)
	queue.Start()

	queue.Enqueue(domain.MailMessage{To: "first@example.com", Subject: "first"})
	queue.Enqueue(domain.MailMessage{To: "second@example.com", Subject: "second"})
	queue.Enqueue(domain.MailMessage{To: "third@example.com", Subject: "third"})

	close(block)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	queue.Shutdown(ctx)
}

type blockingSender struct {
	block chan struct{}
}

func (b *blockingSender) Send(_ context.Context, _ domain.MailMessage) error {
	<-b.block
	return nil
}
