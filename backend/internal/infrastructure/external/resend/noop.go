package resend

import (
	"context"
	"log"

	"github.com/cergijame101007/satehits/internal/domain"
)

// NoOpSender は RESEND_API_KEY 未設定時など、実送信せずログのみ残す
type NoOpSender struct{}

// Send は domain.MailSender を実装する
func (NoOpSender) Send(_ context.Context, msg domain.MailMessage) error {
	log.Printf("mail noop: would send to=%s subject=%q", msg.To, msg.Subject)
	return nil
}
