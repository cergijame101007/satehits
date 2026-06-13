// Package jwt は管理者 API 用アクセストークン（AT）の生成・検証を担う（RT は対象外）
package jwt

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims は AT ペイロード（docs/api_design.md の AT クレーム）
type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// JWTService は管理者向け AT（JWT HS256）の発行・検証
type JWTService struct {
	secret   []byte
	issuer   string        // "satehits-api"
	audience string        // "satehits-admin"
	ttl      time.Duration // 1 hour
}

// NewJWTService は AT 用の JWTService を生成する
func NewJWTService(secret []byte, issuer string, audience string, ttl time.Duration) *JWTService {
	return &JWTService{
		secret:   secret,
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
	}
}

// GenerateAccessToken は AT を発行する
// userID と role は AT 契約どおりか検証する。email はクレームに載せるのみ（形式・存在確認はログイン usecase）
func (s *JWTService) GenerateAccessToken(userID int64, email string, role string) (string, time.Time, error) {
	if err := validateUserID(userID); err != nil {
		return "", time.Time{}, fmt.Errorf("invalid user id (subject): %w", err)
	}
	if err := validateRole(role); err != nil {
		return "", time.Time{}, fmt.Errorf("invalid role: %w", err)
	}
	now := time.Now()
	expiresAt := now.Add(s.ttl)
	claims := Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   strconv.FormatInt(userID, 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, expiresAt, nil
}

// VerifyAccessToken は AT を検証する
// 署名・alg・iss・aud・時刻クレームに加え sub（admin_users.id）と role（owner / developer）を検証する
// email は検証しない。API ごとの認可は usecase / handler 側
func (s *JWTService) VerifyAccessToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithAudience(s.audience),
		jwt.WithExpirationRequired(),
		jwt.WithNotBeforeRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token: invalid signature")
	}

	if claims.IssuedAt == nil {
		return nil, fmt.Errorf("invalid token: missing iat")
	}
	if err := validateSubject(claims.Subject); err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if err := validateRole(claims.Role); err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return claims, nil
}

// validateUserID は admin_users.id が正の整数であることを確認する
func validateUserID(userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("non-positive user id: %d", userID)
	}
	return nil
}

// validateSubject は sub が admin_users.id として正の整数文字列か確認する
func validateSubject(sub string) error {
	if sub == "" {
		return fmt.Errorf("empty subject")
	}
	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return fmt.Errorf("non-numeric subject: %w", err)
	}
	return validateUserID(id)
}

// validateRole は role が owner / developer か確認する
func validateRole(role string) error {
	if role == "" {
		return fmt.Errorf("empty role")
	}
	if role != "owner" && role != "developer" {
		return fmt.Errorf("invalid role: %q", role)
	}
	return nil
}
