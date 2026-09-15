package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/cergijame101007/satehits/internal/application"
	authusecase "github.com/cergijame101007/satehits/internal/application/usecase/auth"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/pkg/jwt"
)

const (
	testAuthOrigin      = "http://localhost:4321"
	testAuthIssuer      = "satehits-api"
	testAuthAudience    = "satehits-admin"
	testAuthRefreshPath = "/api/v1/admin/refresh"
	testAuthLogoutPath  = "/api/v1/admin/logout"
	testAuthEmail       = "owner@example.com"
	// Cookie に載せる RT 平文（generateRefreshToken と同じ 64 hex）
	testAuthRTPlain = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

var (
	// RT 平文・RT ハッシュ・SHA-256 は 64 hex、AT は base64url の JWT（先頭 "eyJ"）、bcrypt は "$2a$" 等で始まる
	reHex64       = regexp.MustCompile(`[0-9a-f]{64}`)
	reJWT         = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	reBcryptHash  = regexp.MustCompile(`\$2[aby]\$\d\d\$`)
	testAuthRTNow = time.Now()
)

type authTestTxManager struct {
	calls int
	fail  error
}

func (s *authTestTxManager) DoInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	s.calls++
	if s.fail != nil {
		return s.fail
	}
	return fn(ctx)
}

var _ application.TxManager = (*authTestTxManager)(nil)

// authTestDeps は AuthHandler の依存。nil のフィールドは既定の fake で埋める
type authTestDeps struct {
	admin        *authTestAdminRepo
	refresh      *authTestRefreshRepo
	attempts     *authTestLoginAttemptRepo
	tx           *authTestTxManager
	corsOrigins  []string
	cookieDomain string
}

func authTestJWTService() *jwt.JWTService {
	return jwt.NewJWTService([]byte(testAuthJWTSecret), testAuthIssuer, testAuthAudience, time.Hour)
}

func newAuthHandlerWithDeps(t *testing.T, deps authTestDeps) *AuthHandler {
	t.Helper()
	if deps.admin == nil {
		deps.admin = &authTestAdminRepo{}
	}
	if deps.refresh == nil {
		deps.refresh = &authTestRefreshRepo{}
	}
	if deps.attempts == nil {
		deps.attempts = &authTestLoginAttemptRepo{}
	}
	if deps.tx == nil {
		deps.tx = &authTestTxManager{}
	}
	if deps.corsOrigins == nil {
		deps.corsOrigins = []string{testAuthOrigin}
	}
	jwtSvc := authTestJWTService()
	loginUC := authusecase.NewLoginUseCase(deps.admin, deps.refresh, deps.attempts, jwtSvc, authusecase.LoginRateLimitPolicy{
		EmailMax: 5,
		IPMax:    20,
		Window:   15 * time.Minute,
	})
	refreshUC := authusecase.NewRefreshUseCase(deps.admin, deps.refresh, jwtSvc, deps.tx)
	logoutUC := authusecase.NewLogoutUseCase(deps.refresh)
	return NewAuthHandler(loginUC, refreshUC, logoutUC, deps.corsOrigins, deps.cookieDomain, 1)
}

func authTestOwner(t *testing.T) domain.AdminUser {
	t.Helper()
	return domain.AdminUser{
		ID:           1,
		Email:        testAuthEmail,
		PasswordHash: mustAuthPasswordHash(t, testAuthPassword),
		Role:         "owner",
	}
}

func authTestTokenHash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// authTestActiveRT は testAuthRTPlain に対応する有効な RT 行
func authTestActiveRT(adminUserID int64) domain.RefreshToken {
	return domain.RefreshToken{
		ID:          42,
		AdminUserID: adminUserID,
		TokenHash:   authTestTokenHash(testAuthRTPlain),
		ExpiresAt:   testAuthRTNow.Add(24 * time.Hour),
	}
}

func mustAuthAccessToken(t *testing.T, svc *jwt.JWTService, userID int64, email, role string) string {
	t.Helper()
	token, _, err := svc.GenerateAccessToken(userID, email, role)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	return token
}

// mustSignedAuthToken は標準クレームを持つ AT を HS256 で署名する。edit でクレームを崩す
func mustSignedAuthToken(t *testing.T, secret []byte, edit func(c *jwt.Claims)) string {
	t.Helper()
	now := time.Now()
	claims := &jwt.Claims{
		Email: testAuthEmail,
		Role:  "owner",
		RegisteredClaims: gojwt.RegisteredClaims{
			ID:        "test-jti",
			Issuer:    testAuthIssuer,
			Audience:  gojwt.ClaimStrings{testAuthAudience},
			ExpiresAt: gojwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now),
			Subject:   "1",
		},
	}
	if edit != nil {
		edit(claims)
	}
	token, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

// captureLog は log の出力をバッファへ向け、テスト終了時に stderr へ戻す
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

// assertNoSecretsInLog はログにパスワード平文・bcrypt ハッシュ・64 hex（RT 平文 / RT ハッシュ）・JWT が無いことを確認する
func assertNoSecretsInLog(t *testing.T, logText string, plaintexts ...string) {
	t.Helper()
	for _, p := range plaintexts {
		if p != "" && strings.Contains(logText, p) {
			t.Fatalf("log contains secret %q: %s", p, logText)
		}
	}
	if m := reBcryptHash.FindString(logText); m != "" {
		t.Fatalf("log contains bcrypt hash (%s): %s", m, logText)
	}
	if m := reHex64.FindString(logText); m != "" {
		t.Fatalf("log contains 64-hex token or hash (%s): %s", m, logText)
	}
	if m := reJWT.FindString(logText); m != "" {
		t.Fatalf("log contains JWT (%s): %s", m, logText)
	}
}

func decodeJSONObject(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal() err = %v; body = %s", err, body)
	}
	return payload
}

func assertJSONKeys(t *testing.T, payload map[string]json.RawMessage, want ...string) {
	t.Helper()
	if len(payload) != len(want) {
		t.Fatalf("keys = %v, want exactly %v", keysOf(payload), want)
	}
	for _, k := range want {
		if _, ok := payload[k]; !ok {
			t.Fatalf("keys = %v, want %q present", keysOf(payload), k)
		}
	}
}

func keysOf(payload map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	return keys
}

func findRefreshCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshTokenCookieName {
			return c
		}
	}
	return nil
}

func assertNoSetCookie(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if v := rec.Header().Values("Set-Cookie"); len(v) != 0 {
		t.Fatalf("Set-Cookie = %v, want none", v)
	}
}

// assertRefreshCookieAttributes は docs/api_design.md の RT Cookie 仕様（値以外）を確認する
func assertRefreshCookieAttributes(t *testing.T, c *http.Cookie, wantMaxAge int, wantDomain string) {
	t.Helper()
	if c == nil {
		t.Fatal("refresh_token cookie is missing")
	}
	if c.Path != adminCookiePath {
		t.Fatalf("Path = %q, want %q", c.Path, adminCookiePath)
	}
	if c.MaxAge != wantMaxAge {
		t.Fatalf("MaxAge = %d, want %d", c.MaxAge, wantMaxAge)
	}
	if !c.HttpOnly {
		t.Fatal("HttpOnly = false, want true")
	}
	if !c.Secure {
		t.Fatal("Secure = false, want true")
	}
	if c.SameSite != http.SameSiteNoneMode {
		t.Fatalf("SameSite = %v, want None", c.SameSite)
	}
	if c.Domain != wantDomain {
		t.Fatalf("Domain = %q, want %q", c.Domain, wantDomain)
	}
}

func newLoginRequest(email, password string) *http.Request {
	body, _ := json.Marshal(loginRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, testAuthLoginPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", mediaTypeJSON)
	req.RemoteAddr = "203.0.113.11:1234"
	return req
}

// newCookieAuthRequest は refresh / logout 用のリクエスト。origin / cookieValue は空なら付けない
func newCookieAuthRequest(path, origin, cookieValue string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: cookieValue})
	}
	return req
}

func assertStatusAndCode(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, wantStatus, rec.Body.String())
	}
	assertAuthErrorCode(t, rec, wantCode)
}

func assertInt64s(t *testing.T, name string, got, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
}

func assertStrings(t *testing.T, name string, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}
