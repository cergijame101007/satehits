CREATE TABLE login_attempts (
    id BIGSERIAL PRIMARY KEY,
    email_key TEXT NOT NULL,
    ip TEXT NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_attempts_email ON login_attempts(email_key, attempted_at DESC);
CREATE INDEX idx_login_attempts_ip ON login_attempts(ip, attempted_at DESC);
