package mail

import (
	"fmt"
	"strings"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

const (
	storeName        = "さて、羊に戻るとしよう"
	storeAddress     = "〒933-0871 富山県高岡市駅南5丁目4-7"
	storeInstagram   = "@satehits"
	subjectPrefix    = "【さて、羊に戻るとしよう】"
	cancelPolicyText = "キャンセル・変更は前日までに Instagram（@satehits）の DM よりご連絡ください。当日キャンセルはキャンセル料 100% を申し受けます。"
	noReplyFooter    = "※ 本メールは送信専用です。返信いただいてもお答えできません。お問い合わせは Instagram（@satehits）の DM よりお願いいたします。"
)

type reservationMailContent struct {
	Subject string
	HTML    string
	Text    string
}

func buildReservationReceived(r domain.Reservation) reservationMailContent {
	subject := subjectPrefix + "ご予約を受け付けました"
	detail := reservationDetailLines(r)
	body := strings.Join([]string{
		fmt.Sprintf("%s 様", r.Name),
		"",
		fmt.Sprintf("%s へのご予約申請を受け付けました。", storeName),
		"内容を確認のうえ、改めてご連絡いたします。",
		"",
		detail,
		"",
		noReplyFooter,
	}, "\n")
	html := plainToHTML(body)
	return reservationMailContent{Subject: subject, HTML: html, Text: body}
}

func buildReservationApproved(r domain.Reservation) reservationMailContent {
	subject := subjectPrefix + "ご予約が確定しました"
	detail := reservationDetailLines(r)
	body := strings.Join([]string{
		fmt.Sprintf("%s 様", r.Name),
		"",
		fmt.Sprintf("%s へのご予約が確定しました。", storeName),
		"",
		detail,
		"",
		fmt.Sprintf("店舗: %s", storeName),
		fmt.Sprintf("住所: %s", storeAddress),
		"",
		cancelPolicyText,
		"",
		noReplyFooter,
	}, "\n")
	html := plainToHTML(body)
	return reservationMailContent{Subject: subject, HTML: html, Text: body}
}

func buildReservationRejected(r domain.Reservation, reason string) reservationMailContent {
	subject := subjectPrefix + "ご予約についてのお知らせ"
	lines := []string{
		fmt.Sprintf("%s 様", r.Name),
		"",
		fmt.Sprintf("この度は %s へのご予約をお申し込みいただき、ありがとうございました。", storeName),
		"誠に恐れ入りますが、ご希望の日程ではご予約をお受けできませんでした。",
	}
	reason = strings.TrimSpace(reason)
	if reason != "" {
		lines = append(lines, "", fmt.Sprintf("理由：%s", reason))
	}
	lines = append(lines,
		"",
		"別の日程をご希望の場合は、Instagram（"+storeInstagram+"）の DM よりお気軽にご相談ください。",
		"",
		noReplyFooter,
	)
	body := strings.Join(lines, "\n")
	html := plainToHTML(body)
	return reservationMailContent{Subject: subject, HTML: html, Text: body}
}

// buildPendingReminder は UC-S04 オーナー向けリマインド。宛先がオーナーなので個人情報はマスクしない。
// 対象ウィンドウに幅があるため「3 日前」「前日」と断定せず来店日を明記する
func buildPendingReminder(r domain.Reservation, totalPending int, adminURL string) reservationMailContent {
	visit := r.VisitDate.UTC()
	subject := fmt.Sprintf("%s未承認のご予約があります（%d/%d 来店）", subjectPrefix, int(visit.Month()), visit.Day())
	note := strings.TrimSpace(r.Note)
	if note == "" {
		note = "なし"
	}
	body := strings.Join([]string{
		"未承認（pending）の Web 予約があります。管理画面から承認または拒否をお願いします。",
		"",
		"【ご予約内容】",
		fmt.Sprintf("来店日: %s（%s）", r.VisitDate.Format("2006-01-02"), jaWeekday(r.VisitDate.Weekday())),
		fmt.Sprintf("来店時間: %s", r.VisitTime.Format("15:04")),
		fmt.Sprintf("人数: %d 名", r.People),
		fmt.Sprintf("お名前: %s", r.Name),
		fmt.Sprintf("電話番号: %s", r.Phone),
		fmt.Sprintf("メール: %s", r.Email),
		fmt.Sprintf("備考: %s", note),
		"",
		fmt.Sprintf("現在未対応（pending）の予約は全部で %d 件です。", totalPending),
		"",
		fmt.Sprintf("管理画面: %s", adminURL),
	}, "\n")
	html := plainToHTML(body)
	return reservationMailContent{Subject: subject, HTML: html, Text: body}
}

func jaWeekday(w time.Weekday) string {
	return [...]string{"日", "月", "火", "水", "木", "金", "土"}[w]
}

func reservationDetailLines(r domain.Reservation) string {
	return strings.Join([]string{
		"【ご予約内容】",
		fmt.Sprintf("来店日: %s", r.VisitDate.Format("2006-01-02")),
		fmt.Sprintf("来店時間: %s", r.VisitTime.Format("15:04")),
		fmt.Sprintf("人数: %d 名", r.People),
	}, "\n")
}

func plainToHTML(text string) string {
	paragraphs := strings.Split(text, "\n\n")
	var b strings.Builder
	for i, p := range paragraphs {
		if i > 0 {
			b.WriteString("\n")
		}
		escaped := strings.ReplaceAll(p, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")
		b.WriteString("<p>")
		b.WriteString(escaped)
		b.WriteString("</p>")
	}
	return b.String()
}
