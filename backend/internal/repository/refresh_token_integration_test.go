package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// refresh_tokens の期限切れ掃除（DeleteExpired）の結合テスト
// TEST_DATABASE_URL 未設定時は skip（openTestDB 参照）

func insertTestAdminUser(t *testing.T, db *sql.DB, email string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(
		`INSERT INTO admin_users (email, password_hash, role) VALUES ($1, 'x', 'owner') RETURNING id`,
		email,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert admin_users: %v", err)
	}
	t.Cleanup(func() {
		// refresh_tokens は ON DELETE CASCADE で一緒に消える
		_, _ = db.Exec(`DELETE FROM admin_users WHERE id = $1`, id)
	})
	return id
}

func insertTestRefreshToken(t *testing.T, db *sql.DB, adminUserID int64, tokenHash string, expiresAt time.Time, revokedAt *time.Time) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO refresh_tokens (admin_user_id, token_hash, expires_at, revoked_at) VALUES ($1, $2, $3, $4)`,
		adminUserID, tokenHash, expiresAt, revokedAt,
	)
	if err != nil {
		t.Fatalf("insert refresh_tokens: %v", err)
	}
}

func remainingTokenHashes(t *testing.T, db *sql.DB, adminUserID int64) map[string]bool {
	t.Helper()
	rows, err := db.Query(`SELECT token_hash FROM refresh_tokens WHERE admin_user_id = $1`, adminUserID)
	if err != nil {
		t.Fatalf("select refresh_tokens: %v", err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[h] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewPostgresRefreshTokenRepository(db)

	userID := insertTestAdminUser(t, db, "delete-expired@example.com")
	now := time.Now()
	revoked := now.Add(-time.Hour)

	// 期限切れ（未 revoke / revoke 済み）は消え、未期限（未 revoke / revoke 済み）は残る
	insertTestRefreshToken(t, db, userID, "expired-active", now.Add(-time.Minute), nil)
	insertTestRefreshToken(t, db, userID, "expired-revoked", now.Add(-24*time.Hour), &revoked)
	insertTestRefreshToken(t, db, userID, "valid-active", now.Add(24*time.Hour), nil)
	insertTestRefreshToken(t, db, userID, "valid-revoked", now.Add(24*time.Hour), &revoked)

	t.Run("deletes only rows whose expires_at is before now", func(t *testing.T) {
		deleted, err := repo.DeleteExpired(ctx, now)
		if err != nil {
			t.Fatalf("DeleteExpired: %v", err)
		}
		if deleted != 2 {
			t.Fatalf("deleted = %d, want 2", deleted)
		}
		remaining := remainingTokenHashes(t, db, userID)
		for _, want := range []string{"valid-active", "valid-revoked"} {
			if !remaining[want] {
				t.Errorf("token %q was deleted, want kept", want)
			}
		}
		for _, gone := range []string{"expired-active", "expired-revoked"} {
			if remaining[gone] {
				t.Errorf("token %q still exists, want deleted", gone)
			}
		}
	})

	t.Run("returns zero when nothing is expired", func(t *testing.T) {
		deleted, err := repo.DeleteExpired(ctx, now)
		if err != nil {
			t.Fatalf("DeleteExpired: %v", err)
		}
		if deleted != 0 {
			t.Fatalf("deleted = %d, want 0", deleted)
		}
	})
}
