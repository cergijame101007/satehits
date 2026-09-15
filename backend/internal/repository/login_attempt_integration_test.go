package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

// login_attempts の集計窓（CountRecent の since 境界）の結合テスト
// TEST_DATABASE_URL 未設定時は skip（openTestDB 参照）

const loginAttemptTestWindow = 15 * time.Minute

// uniqueLoginAttemptKeys は他テストの行と混ざらないメールキーと IP を返し、終了時に行を消す
func uniqueLoginAttemptKeys(t *testing.T, db *sql.DB) (emailKey, ip string) {
	t.Helper()
	suffix := uuid.NewString()[:8]
	emailKey = "login-attempt-" + suffix + "@example.com"
	ip = "198.51.100." + suffix
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM login_attempts WHERE email_key = $1 OR ip = $2`, emailKey, ip)
	})
	return emailKey, ip
}

func recordLoginFailure(t *testing.T, ctx context.Context, repo *PostgresLoginAttemptRepository, emailKey, ip string, at time.Time) {
	t.Helper()
	if err := repo.RecordFailure(ctx, emailKey, ip, at); err != nil {
		t.Fatalf("RecordFailure(%s): %v", at, err)
	}
}

func TestLoginAttemptRepository_CountRecent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewPostgresLoginAttemptRepository(db)
	// TIMESTAMPTZ はマイクロ秒精度なので比較用に丸める
	now := time.Now().UTC().Truncate(time.Microsecond)

	t.Run("counts attempts inside window and ignores older ones", func(t *testing.T) {
		emailKey, ip := uniqueLoginAttemptKeys(t, db)
		recordLoginFailure(t, ctx, repo, emailKey, ip, now.Add(-20*time.Minute))
		recordLoginFailure(t, ctx, repo, emailKey, ip, now.Add(-10*time.Minute))
		recordLoginFailure(t, ctx, repo, emailKey, ip, now.Add(-1*time.Minute))

		counts, err := repo.CountRecent(ctx, emailKey, ip, now.Add(-loginAttemptTestWindow))
		if err != nil {
			t.Fatalf("CountRecent: %v", err)
		}
		if counts.ByEmail != 2 || counts.ByIP != 2 {
			t.Fatalf("ByEmail = %d, ByIP = %d, want 2 and 2", counts.ByEmail, counts.ByIP)
		}
		wantOldest := now.Add(-10 * time.Minute)
		if !counts.OldestByEmail.Equal(wantOldest) || !counts.OldestByIP.Equal(wantOldest) {
			t.Fatalf("OldestByEmail = %s, OldestByIP = %s, want %s", counts.OldestByEmail, counts.OldestByIP, wantOldest)
		}
	})

	t.Run("counts attempt exactly at window start", func(t *testing.T) {
		emailKey, ip := uniqueLoginAttemptKeys(t, db)
		since := now.Add(-loginAttemptTestWindow)
		recordLoginFailure(t, ctx, repo, emailKey, ip, since)
		recordLoginFailure(t, ctx, repo, emailKey, ip, since.Add(-time.Microsecond))

		counts, err := repo.CountRecent(ctx, emailKey, ip, since)
		if err != nil {
			t.Fatalf("CountRecent: %v", err)
		}
		if counts.ByEmail != 1 || counts.ByIP != 1 {
			t.Fatalf("ByEmail = %d, ByIP = %d, want 1 and 1", counts.ByEmail, counts.ByIP)
		}
	})

	t.Run("counts email and ip independently", func(t *testing.T) {
		emailKey, ip := uniqueLoginAttemptKeys(t, db)
		otherEmail, otherIP := uniqueLoginAttemptKeys(t, db)
		recordLoginFailure(t, ctx, repo, emailKey, ip, now.Add(-5*time.Minute))
		recordLoginFailure(t, ctx, repo, otherEmail, ip, now.Add(-4*time.Minute))
		recordLoginFailure(t, ctx, repo, emailKey, otherIP, now.Add(-3*time.Minute))

		counts, err := repo.CountRecent(ctx, emailKey, ip, now.Add(-loginAttemptTestWindow))
		if err != nil {
			t.Fatalf("CountRecent: %v", err)
		}
		if counts.ByEmail != 2 || counts.ByIP != 2 {
			t.Fatalf("ByEmail = %d, ByIP = %d, want 2 and 2", counts.ByEmail, counts.ByIP)
		}
	})

	t.Run("releases limit after window has passed", func(t *testing.T) {
		emailKey, ip := uniqueLoginAttemptKeys(t, db)
		latest := now.Add(-1 * time.Minute)
		recordLoginFailure(t, ctx, repo, emailKey, ip, now.Add(-10*time.Minute))
		recordLoginFailure(t, ctx, repo, emailKey, ip, latest)

		// 最後の失敗から窓の長さを過ぎた時点の since では、どの行も数えられない
		later := latest.Add(loginAttemptTestWindow + time.Second)
		counts, err := repo.CountRecent(ctx, emailKey, ip, later.Add(-loginAttemptTestWindow))
		if err != nil {
			t.Fatalf("CountRecent: %v", err)
		}
		if counts.ByEmail != 0 || counts.ByIP != 0 {
			t.Fatalf("ByEmail = %d, ByIP = %d, want 0 and 0", counts.ByEmail, counts.ByIP)
		}
		if !counts.OldestByEmail.IsZero() || !counts.OldestByIP.IsZero() {
			t.Fatalf("OldestByEmail = %s, OldestByIP = %s, want zero", counts.OldestByEmail, counts.OldestByIP)
		}
	})
}
