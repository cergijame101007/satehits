package usecase

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	maxSupplierNameRunes        = 100
	maxSupplierDescriptionRunes = 1000
	maxSupplierURLOctets        = 2048
)

// validateSupplierFields は作成・更新で共通の必須・文字数・URL 形式検証を行う。
func validateSupplierFields(name, description string, instagramURL *string) []FieldViolation {
	var violations []FieldViolation

	trimmedName := strings.TrimSpace(name)
	nameRunes := utf8.RuneCountInString(trimmedName)
	if nameRunes == 0 {
		violations = append(violations, FieldViolation{Field: "name", Message: "取引先名は必須です"})
	} else if nameRunes > maxSupplierNameRunes {
		violations = append(violations, FieldViolation{Field: "name", Message: fmt.Sprintf("取引先名は1〜%d文字で入力してください", maxSupplierNameRunes)})
	}

	trimmedDesc := strings.TrimSpace(description)
	descRunes := utf8.RuneCountInString(trimmedDesc)
	if descRunes == 0 {
		violations = append(violations, FieldViolation{Field: "description", Message: "説明文は必須です"})
	} else if descRunes > maxSupplierDescriptionRunes {
		violations = append(violations, FieldViolation{Field: "description", Message: fmt.Sprintf("説明文は1〜%d文字で入力してください", maxSupplierDescriptionRunes)})
	}

	if v := validateOptionalURL("instagram_url", instagramURL); v != nil {
		violations = append(violations, v...)
	}

	return violations
}

// validateOptionalURL は任意の URL フィールドを検証する。nil または空文字はスキップ。
func validateOptionalURL(field string, raw *string) []FieldViolation {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	if len(value) > maxSupplierURLOctets {
		return []FieldViolation{{Field: field, Message: "URLが長すぎます"}}
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return []FieldViolation{{Field: field, Message: "URLの形式が正しくありません"}}
	}
	return nil
}

// trimmedPtr は文字列ポインタを trim し、空文字なら nil を返す（NULL 保存用に正規化）。
func trimmedPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
