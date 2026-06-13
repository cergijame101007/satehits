package usecase

import (
	"net/mail"
	"strings"
)

const (
	// RFC 5321 に基づくメールアドレスの最大長
	maxLoginEmailOctets = 254
)

func validateLogin(cmd LoginCommand) []FieldViolation {
	var violations []FieldViolation

	email := strings.TrimSpace(cmd.Email)
	if email == "" {
		violations = append(violations, FieldViolation{Field: "email", Message: "メールアドレスは必須です"})
	} else if !validLoginEmail(email) {
		violations = append(violations, FieldViolation{Field: "email", Message: "メールアドレスの形式が正しくありません"})
	}

	if cmd.Password == "" {
		violations = append(violations, FieldViolation{Field: "password", Message: "パスワードは必須です"})
	}

	return violations
}

// validLoginEmail は API フィールド用の単純なメール文字列のみ許可する
// 角括弧付きの RFC 5322 形式（例: "田中 <tanaka@example.com>"）や Display Name 付きは拒否する
func validLoginEmail(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if len(s) > maxLoginEmailOctets {
		return false
	}
	if strings.ContainsAny(s, "<>") {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return false
	}
	if addr.Name != "" {
		return false
	}
	return addr.Address == s
}
