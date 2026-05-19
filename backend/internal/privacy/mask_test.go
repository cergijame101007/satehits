package privacy

import "testing"

func TestMaskName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "masks empty string", in: "", want: "*"},
		{name: "masks whitespace only", in: "  ", want: "*"},
		{name: "masks single character", in: "山", want: "*"},
		{name: "masks Japanese full name", in: "山田太郎", want: "山***"},
		{name: "masks Latin full name", in: "John Doe", want: "J*******"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskName(tt.in)
			if got != tt.want {
				t.Fatalf("MaskName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "masks empty string", in: "", want: "***"},
		{name: "masks non-digit input", in: "abc", want: "***"},
		{name: "masks domestic number keeping last four digits", in: "090-1234-5678", want: "***5678"},
		{name: "masks short digit-only input", in: "1234", want: "****"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskPhone(tt.in)
			if got != tt.want {
				t.Fatalf("MaskPhone(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "masks empty string", in: "", want: "***"},
		{name: "masks invalid address", in: "invalid", want: "***"},
		{name: "masks standard email", in: "yamada@example.com", want: "y***@***.com"},
		{name: "masks short local part", in: "a@b.co", want: "a***@***.co"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskEmail(tt.in)
			if got != tt.want {
				t.Fatalf("MaskEmail(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
