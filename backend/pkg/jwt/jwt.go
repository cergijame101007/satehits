package jwt

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secret   []byte
	issuer   string        // "satehits-api"
	audience string        // "satehits-admin"
	ttl      time.Duration // 1 hour
}

func NewJWTService(secret []byte, issuer string, audience string, ttl time.Duration) *JWTService {
	return &JWTService{
		secret:   secret,
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
	}
}

func (s *JWTService) GenerateToken(userID int64, email string, role string) (string, time.Time, error) {
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

func (s *JWTService) VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(
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

// validateSubject は sub（admin_users.id）が空でなく数値であることを確認する
func validateSubject(sub string) error {
	if sub == "" {
		return fmt.Errorf("empty subject")
	}
	if _, err := strconv.ParseInt(sub, 10, 64); err != nil {
		return fmt.Errorf("non-numeric subject: %w", err)
	}
	return nil
}

// validateRole は role が docs の owner / developer のいずれかであることを確認する
func validateRole(role string) error {
	if role == "" {
		return fmt.Errorf("empty role")
	}
	if role != "owner" && role != "developer" {
		return fmt.Errorf("invalid role: %q", role)
	}
	return nil
}
