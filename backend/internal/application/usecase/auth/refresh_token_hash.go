package usecase

import (
	"crypto/sha256"
	"encoding/hex"
)

// hashRefreshTokenPlain はリフレッシュトークンの平文をハッシュ化する
func hashRefreshTokenPlain(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
