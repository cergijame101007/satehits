// Package privacy はログや監査出力向けの個人情報マスキングを提供する。
package privacy

import (
	"strings"
	"unicode/utf8"
)

// MaskName は氏名をマスキングする（先頭1文字のみ残す。1文字のみの場合は *）。
func MaskName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "*"
	}
	rn := utf8.RuneCountInString(name)
	if rn == 1 {
		return "*"
	}
	r, size := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError && size == 1 {
		return "*"
	}
	return string(r) + strings.Repeat("*", rn-1)
}

// MaskPhone は電話番号をマスキングする（数字の下4桁のみ残す）。
func MaskPhone(phone string) string {
	var b strings.Builder
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	digits := b.String()
	n := len(digits)
	if n == 0 {
		return "***"
	}
	if n <= 4 {
		return "****"
	}
	return "***" + digits[n-4:]
}

// MaskEmail はメールアドレスをマスキングする（ローカル先頭1文字 + ドメインはTLDのみ）。
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 || at >= len(email)-1 {
		return "***"
	}
	local, domain := email[:at], email[at+1:]
	if domain == "" {
		return "***"
	}
	first, size := utf8.DecodeRuneInString(local)
	maskedLocal := "*"
	if size > 0 && first != utf8.RuneError {
		maskedLocal = string(first) + "***"
	}
	lastDot := strings.LastIndex(domain, ".")
	if lastDot > 0 && lastDot < len(domain)-1 {
		return maskedLocal + "@***" + domain[lastDot:]
	}
	return maskedLocal + "@***"
}
