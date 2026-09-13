package usecase

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cergijame101007/satehits/internal/domain"
)

// admin_users.password_hash の bcrypt cost（docs/table_design.md）
const documentedPasswordHashCost = 12

func mustPasswordHashWithCost(t *testing.T, plain string, cost int) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword(cost=%d): %v", cost, err)
	}
	return string(hash)
}

func TestDummyPasswordHash(t *testing.T) {
	t.Run("is a bcrypt hash at the documented cost", func(t *testing.T) {
		if !strings.HasPrefix(string(dummyPasswordHash), "$2a$") {
			t.Fatalf("dummyPasswordHash = %q, want bcrypt $2a$ prefix", dummyPasswordHash)
		}
		cost, err := bcrypt.Cost(dummyPasswordHash)
		if err != nil {
			t.Fatalf("bcrypt.Cost: %v", err)
		}
		if cost != documentedPasswordHashCost {
			t.Fatalf("dummyPasswordHash cost = %d, want %d", cost, documentedPasswordHashCost)
		}
	})

	t.Run("does not match common passwords", func(t *testing.T) {
		for _, pw := range []string{testLoginPassword, ""} {
			if err := bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(pw)); err == nil {
				t.Fatalf("dummyPasswordHash matches %q, want mismatch", pw)
			}
		}
	})
}

func TestLoginUseCase_Execute_comparesPasswordWithBcrypt(t *testing.T) {
	t.Run("rejects when stored hash equals the plaintext password", func(t *testing.T) {
		admin := &fakeAdminUserRepo{user: domain.AdminUser{
			ID:           1,
			Email:        testLoginEmail,
			PasswordHash: testLoginPassword,
			Role:         "owner",
		}}
		uc := newTestLoginUseCase(t, admin, &fakeRefreshTokenRepo{}, &fakeLoginAttemptRepo{})

		result, err := uc.Execute(context.Background(), validLoginCommand())

		assertExecuteError(t, err, domain.ErrAdminUserUnauthorized, "")
		if result != nil {
			t.Fatalf("result = %#v, want nil", result)
		}
	})

	t.Run("accepts a bcrypt hash generated at the documented cost", func(t *testing.T) {
		admin := &fakeAdminUserRepo{user: domain.AdminUser{
			ID:           1,
			Email:        testLoginEmail,
			PasswordHash: mustPasswordHashWithCost(t, testLoginPassword, documentedPasswordHashCost),
			Role:         "owner",
		}}
		uc := newTestLoginUseCase(t, admin, &fakeRefreshTokenRepo{}, &fakeLoginAttemptRepo{})

		result, err := uc.Execute(context.Background(), validLoginCommand())
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if result == nil || result.AccessToken == "" {
			t.Fatalf("result = %#v, want access token", result)
		}
	})
}

// 未登録メールと登録済みメールの誤パスワードで所要時間が揃うことを、中央値の比で確認する
// cost はテスト時間短縮のため 10 に下げ、ダミーハッシュも同じ cost に差し替える（cost が揃うことが前提の仕組みなので、実値の cost は TestDummyPasswordHash で別途確認）
func TestLoginUseCase_Execute_unknownEmailTakesAsLongAsWrongPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("timing measurement is skipped in short mode")
	}

	const (
		cost     = 10
		runs     = 15
		maxRatio = 3.0
		// bcrypt 比較が両パスで実際に走ったことの下限（cost 10 は数十 ms）
		minMedian = 5 * time.Millisecond
	)

	original := dummyPasswordHash
	dummyPasswordHash = []byte(mustPasswordHashWithCost(t, "timing-dummy", cost))
	t.Cleanup(func() { dummyPasswordHash = original })

	owner := domain.AdminUser{
		ID:           1,
		Email:        testLoginEmail,
		PasswordHash: mustPasswordHashWithCost(t, testLoginPassword, cost),
		Role:         "owner",
	}
	wrongPassword := newTestLoginUseCase(t, &fakeAdminUserRepo{user: owner}, &fakeRefreshTokenRepo{}, &fakeLoginAttemptRepo{})
	unknownEmail := newTestLoginUseCase(t, &fakeAdminUserRepo{findErr: domain.ErrAdminUserNotFound}, &fakeRefreshTokenRepo{}, &fakeLoginAttemptRepo{})
	cmd := LoginCommand{Email: testLoginEmail, Password: "wrong-password", ClientIP: testClientIP}

	execute := func(uc *LoginUseCase) time.Duration {
		start := time.Now()
		if _, err := uc.Execute(context.Background(), cmd); err == nil {
			t.Fatal("Execute err = nil, want unauthorized")
		}
		return time.Since(start)
	}

	execute(wrongPassword)
	execute(unknownEmail)

	wrongTimes := make([]time.Duration, 0, runs)
	unknownTimes := make([]time.Duration, 0, runs)
	for i := 0; i < runs; i++ {
		wrongTimes = append(wrongTimes, execute(wrongPassword))
		unknownTimes = append(unknownTimes, execute(unknownEmail))
	}

	wrongMedian := medianDuration(wrongTimes)
	unknownMedian := medianDuration(unknownTimes)
	if wrongMedian < minMedian || unknownMedian < minMedian {
		t.Fatalf("median wrong=%v unknown=%v, want both >= %v (bcrypt compare must run on both paths)", wrongMedian, unknownMedian, minMedian)
	}

	ratio := float64(wrongMedian) / float64(unknownMedian)
	if ratio < 1 {
		ratio = 1 / ratio
	}
	if ratio > maxRatio {
		t.Fatalf("median wrong=%v unknown=%v ratio=%.2f, want <= %.1f", wrongMedian, unknownMedian, ratio, maxRatio)
	}
}

func medianDuration(ds []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), ds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2]
}
