CREATE TABLE sessions (
    id              UUID        PRIMARY KEY,
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_hash    TEXT        NOT NULL,
    user_agent      TEXT,
    ip              INET,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at      TIMESTAMPTZ
);

-- Fast lookup by user for ListActiveSessions and revoke-all operations.
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- Fast lookup by hash for RefreshToken and Logout.
CREATE INDEX sessions_refresh_hash_idx ON sessions (refresh_hash);

-- Partial index over active sessions only — used by ListActiveForUser.
CREATE INDEX sessions_active_idx ON sessions (user_id, expires_at)
    WHERE revoked_at IS NULL;
