package usecase

import (
	"strings"
	"testing"
)

// validLoginCommand は管理者ログイン API の正常系ベース
func validLoginCommand() LoginCommand {
	return LoginCommand{
		Email:    testLoginEmail,
		Password: testLoginPassword,
		ClientIP: testClientIP,
	}
}

func assertNoViolations(t *testing.T, violations []FieldViolation) {
	t.Helper()
	if len(violations) != 0 {
		t.Fatalf("violations = %#v, want none", violations)
	}
}

func assertSingleViolationField(t *testing.T, violations []FieldViolation, field string) {
	t.Helper()
	if len(violations) != 1 {
		t.Fatalf("violations count = %d, want 1; violations = %#v", len(violations), violations)
	}
	if violations[0].Field != field {
		t.Fatalf("violations[0].Field = %q, want %q; violations = %#v", violations[0].Field, field, violations)
	}
}

func TestValidLoginEmail(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "accepts bare address", in: "a@example.com", want: true},
		{name: "accepts address with plus tag", in: "user.name+tag@example.co.jp", want: true},
		{name: "rejects display-name form", in: `田中 <tanaka@example.com>`, want: false},
		{name: "rejects stray angle bracket", in: "tanaka@example.com>", want: false},
		{name: "rejects string without @", in: "notanemail", want: false},
		{name: "rejects over RFC 5321 max length", in: strings.Repeat("a", 250) + "@x.co", want: false},
		{name: "rejects empty string", in: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validLoginEmail(tt.in)
			if got != tt.want {
				t.Fatalf("validLoginEmail(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateLogin(t *testing.T) {
	t.Run("accepts valid command", func(t *testing.T) {
		assertNoViolations(t, validateLogin(validLoginCommand()))
	})

	t.Run("rejects whitespace-only email", func(t *testing.T) {
		cmd := validLoginCommand()
		cmd.Email = " \t　 "
		assertSingleViolationField(t, validateLogin(cmd), "email")
	})

	t.Run("rejects invalid email format", func(t *testing.T) {
		cmd := validLoginCommand()
		cmd.Email = "not-an-email"
		assertSingleViolationField(t, validateLogin(cmd), "email")
	})

	t.Run("rejects empty password", func(t *testing.T) {
		cmd := validLoginCommand()
		cmd.Password = ""
		assertSingleViolationField(t, validateLogin(cmd), "password")
	})

	t.Run("accepts whitespace-only password for auth layer", func(t *testing.T) {
		cmd := validLoginCommand()
		cmd.Password = " \t　 "
		assertNoViolations(t, validateLogin(cmd))
	})

	t.Run("returns two violations when both email and password are empty", func(t *testing.T) {
		violations := validateLogin(LoginCommand{Email: "", Password: ""})
		if len(violations) != 2 {
			t.Fatalf("violations count = %d, want 2; violations = %#v", len(violations), violations)
		}
		seen := make(map[string]bool)
		for _, v := range violations {
			seen[v.Field] = true
		}
		for _, field := range []string{"email", "password"} {
			if !seen[field] {
				t.Errorf("missing violation for field %q", field)
			}
		}
	})
}
