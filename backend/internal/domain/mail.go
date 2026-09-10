package domain

import (
	"context"
	"errors"
)

// ErrMailPermanent は再送しても成功しない送信エラー（宛先不正など 4xx）
var ErrMailPermanent = errors.New("permanent mail send failure")

// ErrMailAuth は送信 API の認証・認可エラー（401 / 403）。
// メッセージ起因ではなく設定起因なので、行の試行回数を消費せずバッチを中断する
var ErrMailAuth = errors.New("mail sender authentication failure")

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
