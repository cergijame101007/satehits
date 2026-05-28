package jwt

import (
	"encoding/base64"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "satehits-api"
	testAudience = "satehits-admin"
)

// 32 バイト以上の固定 secret（本番値は使わない）
var testSecret = []byte("development-jwt-secret-minimum-32-bytes-long")

func testJWTService(t *testing.T, ttl time.Duration) *JWTService {
	t.Helper()
	return NewJWTService(testSecret, testIssuer, testAudience, ttl)
}

func TestJWTService_GenerateAndVerify(t *testing.T) {
	svc := testJWTService(t, time.Hour)

	token, expiresAt, err := svc.GenerateToken(42, "owner@example.com", "owner")
	if err != nil {
		t.Fatalf("GenerateToken() err = %v, want nil", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() token is empty")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatalf("expiresAt = %v, want after now", expiresAt)
	}

	claims, err := svc.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() err = %v, want nil", err)
	}
	if got, want := claims.Subject, "42"; got != want {
		t.Fatalf("Subject = %q, want %q", got, want)
	}
	if got, want := claims.Email, "owner@example.com"; got != want {
		t.Fatalf("Email = %q, want %q", got, want)
	}
	if got, want := claims.Role, "owner"; got != want {
		t.Fatalf("Role = %q, want %q", got, want)
	}
	if claims.Issuer != testIssuer {
		t.Fatalf("Issuer = %q, want %q", claims.Issuer, testIssuer)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != testAudience {
		t.Fatalf("Audience = %v, want %q", claims.Audience, testAudience)
	}
}

func TestJWTService_GenerateAndVerify_developerRole(t *testing.T) {
	svc := testJWTService(t, time.Hour)

	token, _, err := svc.GenerateToken(2, "dev@example.com", "developer")
	if err != nil {
		t.Fatalf("GenerateToken() err = %v, want nil", err)
	}

	claims, err := svc.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() err = %v, want nil", err)
	}
	if got, want := claims.Role, "developer"; got != want {
		t.Fatalf("Role = %q, want %q", got, want)
	}
}

func TestJWTService_GenerateAndVerify_maxUserID(t *testing.T) {
	svc := testJWTService(t, time.Hour)
	maxID := int64(math.MaxInt64)
	wantSub := strconv.FormatInt(maxID, 10)

	token, _, err := svc.GenerateToken(maxID, "owner@example.com", "owner")
	if err != nil {
		t.Fatalf("GenerateToken() err = %v, want nil", err)
	}

	claims, err := svc.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() err = %v, want nil", err)
	}
	if got, want := claims.Subject, wantSub; got != want {
		t.Fatalf("Subject = %q, want %q", got, want)
	}
}

func TestJWTService_GenerateToken_rejectsInvalidRole(t *testing.T) {
	svc := testJWTService(t, time.Hour)

	tests := []struct {
		name string
		role string
	}{
		{name: "empty role", role: ""},
		{name: "admin role", role: "admin"},
		{name: "uppercase owner", role: "OWNER"},
		{name: "mixed case developer", role: "Developer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expiresAt, err := svc.GenerateToken(1, "owner@example.com", tt.role)
			if err == nil {
				t.Fatal("GenerateToken() err = nil, want error")
			}
			if token != "" {
				t.Fatalf("token = %q, want empty", token)
			}
			if !expiresAt.IsZero() {
				t.Fatalf("expiresAt = %v, want zero", expiresAt)
			}
		})
	}
}

func TestJWTService_GenerateToken_rejectsInvalidUserID(t *testing.T) {
	svc := testJWTService(t, time.Hour)

	tests := []struct {
		name   string
		userID int64
	}{
		{name: "zero user id", userID: 0},
		{name: "negative user id", userID: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expiresAt, err := svc.GenerateToken(tt.userID, "owner@example.com", "owner")
			if err == nil {
				t.Fatal("GenerateToken() err = nil, want error")
			}
			if token != "" {
				t.Fatalf("token = %q, want empty", token)
			}
			if !expiresAt.IsZero() {
				t.Fatalf("expiresAt = %v, want zero", expiresAt)
			}
		})
	}
}

func TestJWTService_VerifyToken_rejectsInvalid(t *testing.T) {
	good := testJWTService(t, time.Hour)
	validToken, _, err := good.GenerateToken(1, "owner@example.com", "owner")
	if err != nil {
		t.Fatalf("setup GenerateToken() err = %v", err)
	}

	tests := []struct {
		name       string
		token      string
		verifyWith *JWTService
	}{
		{
			name:       "rejects malformed string",
			token:      "not-a-jwt",
			verifyWith: good,
		},
		{
			name:       "rejects wrong issuer",
			token:      mustTokenWithWrongIssuer(t),
			verifyWith: good,
		},
		{
			name:       "rejects wrong audience",
			token:      mustTokenWithWrongAudience(t),
			verifyWith: good,
		},
		{
			name:       "rejects missing issuer",
			token:      mustTokenMissingIssuer(t),
			verifyWith: good,
		},
		{
			name:       "rejects missing audience",
			token:      mustTokenMissingAudience(t),
			verifyWith: good,
		},
		{
			name: "rejects token signed with wrong secret",
			token: func() string {
				other := NewJWTService([]byte("another-secret-at-least-32-bytes-xx"), testIssuer, testAudience, time.Hour)
				tok, _, err := other.GenerateToken(1, "owner@example.com", "owner")
				if err != nil {
					t.Fatalf("setup other GenerateToken() err = %v", err)
				}
				return tok
			}(),
			verifyWith: good,
		},
		{
			name:       "rejects none algorithm",
			token:      mustNoneAlgorithmToken(t),
			verifyWith: good,
		},
		{
			name:       "rejects wrong signing method HS512",
			token:      mustTokenHS512(t),
			verifyWith: good,
		},
		{
			name:       "rejects missing exp",
			token:      mustTokenMissingExp(t),
			verifyWith: good,
		},
		{
			name:       "rejects missing nbf",
			token:      mustTokenMissingNBF(t),
			verifyWith: good,
		},
		{
			name:       "rejects missing iat",
			token:      mustTokenMissingIAT(t),
			verifyWith: good,
		},
		{
			name:       "rejects iat in the future",
			token:      mustTokenFutureIAT(t),
			verifyWith: good,
		},
		{
			name:       "rejects nbf in the future",
			token:      mustTokenFutureNBF(t),
			verifyWith: good,
		},
		{
			name:       "rejects empty subject",
			token:      mustTokenWithSubject(t, ""),
			verifyWith: good,
		},
		{
			name:       "rejects non-numeric subject",
			token:      mustTokenWithSubject(t, "not-a-number"),
			verifyWith: good,
		},
		{
			name:       "rejects zero subject",
			token:      mustTokenWithSubject(t, "0"),
			verifyWith: good,
		},
		{
			name:       "rejects negative subject",
			token:      mustTokenWithSubject(t, "-1"),
			verifyWith: good,
		},
		{
			name:       "rejects subject with leading space",
			token:      mustTokenWithSubject(t, " 1"),
			verifyWith: good,
		},
		{
			name:       "rejects subject with trailing space",
			token:      mustTokenWithSubject(t, "1 "),
			verifyWith: good,
		},
		{
			name:       "rejects invalid role",
			token:      mustTokenWithRole(t, "admin"),
			verifyWith: good,
		},
		{
			name:       "rejects empty role",
			token:      mustTokenWithRole(t, ""),
			verifyWith: good,
		},
		{
			name:       "rejects tampered payload",
			token:      mustTamperedToken(t, validToken),
			verifyWith: good,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.verifyWith.VerifyToken(tt.token)
			if err == nil {
				t.Fatal("VerifyToken() err = nil, want error")
			}
		})
	}
}

func TestJWTService_VerifyToken_rejectsExpired(t *testing.T) {
	// 負の TTL で発行時点から期限切れにし、time.Sleep に依存しない
	svc := testJWTService(t, -time.Hour)

	token, _, err := svc.GenerateToken(1, "owner@example.com", "owner")
	if err != nil {
		t.Fatalf("GenerateToken() err = %v", err)
	}

	_, err = svc.VerifyToken(token)
	if err == nil {
		t.Fatal("VerifyToken() err = nil, want error for expired token")
	}
}

// --- helpers（テスト専用）---

func mustTokenWithWrongIssuer(t *testing.T) string {
	t.Helper()
	evil := NewJWTService(testSecret, "evil-issuer", testAudience, time.Hour)
	tok, _, err := evil.GenerateToken(1, "owner@example.com", "owner")
	if err != nil {
		t.Fatalf("GenerateToken() err = %v", err)
	}
	return tok
}

func mustTokenWithWrongAudience(t *testing.T) string {
	t.Helper()
	evil := NewJWTService(testSecret, testIssuer, "evil-audience", time.Hour)
	tok, _, err := evil.GenerateToken(1, "owner@example.com", "owner")
	if err != nil {
		t.Fatalf("GenerateToken() err = %v", err)
	}
	return tok
}

// docs: alg=HS256 のみ許可 → none は拒否すること
func mustNoneAlgorithmToken(t *testing.T) string {
	t.Helper()
	now := time.Now()
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub":   "1",
		"email": "owner@example.com",
		"role":  "owner",
		"iss":   testIssuer,
		"aud":   testAudience,
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
	})
	// v5: none 署名は明示許可が必要（テストでのみ使用）
	token, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build none token: %v", err)
	}
	if !strings.Contains(token, ".") {
		t.Fatal("none token looks malformed")
	}
	return token
}

func mustSignedToken(t *testing.T, edit func(*Claims)) string {
	t.Helper()
	now := time.Now()
	claims := &Claims{
		Email: "owner@example.com",
		Role:  "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "test-jti",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   "1",
		},
	}
	edit(claims)
	unsigned := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := unsigned.SignedString(testSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func mustTokenMissingIssuer(t *testing.T) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.Issuer = ""
	})
}

func mustTokenMissingAudience(t *testing.T) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.Audience = nil
	})
}

func mustTokenMissingExp(t *testing.T) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.ExpiresAt = nil
	})
}

func mustTokenMissingNBF(t *testing.T) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.NotBefore = nil
	})
}

func mustTokenMissingIAT(t *testing.T) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.IssuedAt = nil
	})
}

func mustTokenFutureIAT(t *testing.T) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.IssuedAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	})
}

func mustTokenHS512(t *testing.T) string {
	t.Helper()
	now := time.Now()
	claims := &Claims{
		Email: "owner@example.com",
		Role:  "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "test-jti",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   "1",
		},
	}
	unsigned := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	token, err := unsigned.SignedString(testSecret)
	if err != nil {
		t.Fatalf("sign HS512 token: %v", err)
	}
	return token
}

func mustTokenWithRole(t *testing.T, role string) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.Role = role
	})
}

func mustTokenFutureNBF(t *testing.T) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour))
	})
}

func mustTokenWithSubject(t *testing.T, sub string) string {
	t.Helper()
	return mustSignedToken(t, func(c *Claims) {
		c.Subject = sub
	})
}

func mustTamperedToken(t *testing.T, valid string) string {
	t.Helper()
	parts := strings.Split(valid, ".")
	if len(parts) != 3 {
		t.Fatalf("valid token has %d parts, want 3", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("payload is empty")
	}
	raw[0] ^= 0xff
	parts[1] = base64.RawURLEncoding.EncodeToString(raw)
	return strings.Join(parts, ".")
}
