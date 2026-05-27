package jwt

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
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

	// 新しいトークンを生成して署名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 署名されたトークン文字列を取得
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, expiresAt, nil
}

func (s *JWTService) VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	// トークンを検証
	token, err := jwt.ParseWithClaims(
		tokenString, 
		claims, 
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.secret, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

    // クレームを取得して検証
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        // もし追加の検証が必要なら、ロジックをここに実装
        // 例：ブラックリストチェックなど
        return claims, nil
    }
    return nil, fmt.Errorf("invalid token")
}