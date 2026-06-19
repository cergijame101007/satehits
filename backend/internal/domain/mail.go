package domain

import "context"

// MailMessage は Resend 等へ送るメール 1 通分の内容
type MailMessage struct {
	From    string
	To      string
	Subject string
	HTML    string
	Text    string
}

// MailSender は外部メール API への送信を抽象化する
type MailSender interface {
	Send(ctx context.Context, msg MailMessage) error
}
