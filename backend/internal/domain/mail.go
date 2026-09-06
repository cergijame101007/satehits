package domain

import (
	"context"
	"errors"
)

// ErrMailPermanent は再送しても成功しない送信エラー（4xx 等）
var ErrMailPermanent = errors.New("permanent mail send failure")

// MailMessage は外部メール API へ送る 1 通分の内容
type MailMessage struct {
	From           string
	To             string
	Subject        string
	HTML           string
	Text           string
	IdempotencyKey string
}

// MailSender は外部メール API への送信を抽象化する（Resend 非依存）
type MailSender interface {
	Send(ctx context.Context, msg MailMessage) error
}
