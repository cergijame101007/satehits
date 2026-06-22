package usecase

import (
	"strings"
	"testing"
)

func strptr(s string) *string { return &s }

func assertNoViolations(t *testing.T, violations []FieldViolation) {
	t.Helper()
	if len(violations) > 0 {
		t.Errorf("got violations, want none: %v", violations)
	}
}

func assertHasViolationField(t *testing.T, violations []FieldViolation, field string) {
	t.Helper()
	for _, v := range violations {
		if v.Field == field {
			return
		}
	}
	t.Errorf("got no violation for field %q, want one: %v", field, violations)
}

func TestValidateSupplierFields(t *testing.T) {
	tests := []struct {
		name         string
		supplierName string
		description  string
		instagramURL *string
		wantField    string
	}{
		{name: "accepts valid required fields", supplierName: "〇〇農園", description: "無農薬野菜の生産者です。"},
		{name: "accepts valid instagram url", supplierName: "〇〇農園", description: "説明", instagramURL: strptr("https://instagram.com/example")},
		{name: "accepts nil instagram url", supplierName: "〇〇農園", description: "説明", instagramURL: nil},
		{name: "accepts empty instagram url as unset", supplierName: "〇〇農園", description: "説明", instagramURL: strptr("  ")},
		{name: "rejects empty name", supplierName: "  ", description: "説明", wantField: "name"},
		{name: "rejects too long name", supplierName: strings.Repeat("あ", maxSupplierNameRunes+1), description: "説明", wantField: "name"},
		{name: "accepts boundary name length", supplierName: strings.Repeat("あ", maxSupplierNameRunes), description: "説明"},
		{name: "rejects empty description", supplierName: "〇〇農園", description: "   ", wantField: "description"},
		{name: "rejects too long description", supplierName: "〇〇農園", description: strings.Repeat("あ", maxSupplierDescriptionRunes+1), wantField: "description"},
		{name: "rejects non-http instagram url", supplierName: "〇〇農園", description: "説明", instagramURL: strptr("ftp://example.com"), wantField: "instagram_url"},
		{name: "rejects malformed instagram url", supplierName: "〇〇農園", description: "説明", instagramURL: strptr("not a url"), wantField: "instagram_url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validateSupplierFields(tt.supplierName, tt.description, tt.instagramURL)
			if tt.wantField == "" {
				assertNoViolations(t, violations)
				return
			}
			assertHasViolationField(t, violations, tt.wantField)
		})
	}
}

func TestValidateReorder(t *testing.T) {
	tests := []struct {
		name    string
		ids     []int64
		wantErr bool
	}{
		{name: "accepts unique positive ids", ids: []int64{3, 1, 2}},
		{name: "rejects empty list", ids: []int64{}, wantErr: true},
		{name: "rejects duplicate ids", ids: []int64{1, 2, 1}, wantErr: true},
		{name: "rejects non-positive id", ids: []int64{1, 0, 2}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validateReorder(tt.ids)
			if tt.wantErr {
				assertHasViolationField(t, violations, "order")
				return
			}
			assertNoViolations(t, violations)
		})
	}
}
