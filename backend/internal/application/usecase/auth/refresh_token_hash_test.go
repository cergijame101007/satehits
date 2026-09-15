package usecase

import "testing"

func TestHashRefreshTokenPlain(t *testing.T) {
	const plain = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	t.Run("returns 64 lowercase hex chars that differ from the plaintext", func(t *testing.T) {
		got := hashRefreshTokenPlain(plain)
		if len(got) != 64 {
			t.Fatalf("len = %d, want 64", len(got))
		}
		if got == plain {
			t.Fatal("hash equals plaintext, want SHA-256 digest")
		}
		for _, c := range got {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Fatalf("hash %q contains non-hex char %q", got, c)
			}
		}
		assertRefreshTokenHash(t, plain, got)
	})

	t.Run("is deterministic and distinguishes different tokens", func(t *testing.T) {
		first := hashRefreshTokenPlain(plain)
		second := hashRefreshTokenPlain(plain)
		if first != second {
			t.Fatalf("same plaintext produced different hashes: %q vs %q", first, second)
		}
		if other := hashRefreshTokenPlain(plain[:63] + "0"); other == first {
			t.Fatal("different plaintexts produced the same hash")
		}
	})
}
