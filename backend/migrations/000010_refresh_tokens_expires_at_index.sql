-- 期限切れ refresh_tokens の削除（ログイン成功時のベストエフォート掃除）用インデックス
-- DELETE FROM refresh_tokens WHERE expires_at < $1 の範囲走査を支える
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);
