package usecase

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashRefreshTokenPlain はリフレッシュトークンの平文をハッシュ化する
func HashRefreshTokenPlain(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
