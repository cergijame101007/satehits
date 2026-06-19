package usecase

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cergijame101007/satehits/internal/domain"
)

const maxRejectReasonRunes = 500

func isValidEnum(value string, allowed []string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

func validateReservationStatus(value string) []FieldViolation {
	if value == "" {
		return nil
	}
	if !isValidEnum(value, domain.ValidReservationStatuses) {
		return []FieldViolation{{Field: "status", Message: "ステータスの値が不正です"}}
	}
	return nil
}

func validateReservationSource(value string) []FieldViolation {
	value = strings.TrimSpace(value)
	if value == "" {
		return []FieldViolation{{Field: "source", Message: "予約経路は必須です"}}
	}
	if !isValidEnum(value, domain.ValidReservationSources) {
		return []FieldViolation{{Field: "source", Message: "予約経路の値が不正です"}}
	}
	return nil
}

func validateUpdateStatusTarget(value string) []FieldViolation {
	value = strings.TrimSpace(value)
	if value == "" {
		return []FieldViolation{{Field: "status", Message: "ステータスは必須です"}}
	}
	if !isValidEnum(value, domain.UpdateStatusTargets) {
		return []FieldViolation{{Field: "status", Message: "ステータスの値が不正です"}}
	}
	return nil
}

func validateUpdateStatusReason(reason string) []FieldViolation {
	if utf8.RuneCountInString(reason) > maxRejectReasonRunes {
		return []FieldViolation{
			{Field: "reason", Message: fmt.Sprintf("理由は%d文字以内で入力してください", maxRejectReasonRunes)},
		}
	}
	return nil
}
