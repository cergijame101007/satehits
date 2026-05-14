package privacy

import "testing"

func TestMaskName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", "*"},
		{"  ", "*"},
		{"山", "*"},
		{"山田太郎", "山***"},
		{"John Doe", "J*******"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := MaskName(tt.in); got != tt.want {
				t.Errorf("MaskName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", "***"},
		{"abc", "***"},
		{"090-1234-5678", "***5678"},
		{"1234", "****"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := MaskPhone(tt.in); got != tt.want {
				t.Errorf("MaskPhone(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", "***"},
		{"invalid", "***"},
		{"yamada@example.com", "y***@***.com"},
		{"a@b.co", "a***@***.co"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := MaskEmail(tt.in); got != tt.want {
				t.Errorf("MaskEmail(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
