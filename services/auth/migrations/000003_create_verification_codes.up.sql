CREATE TABLE verification_codes (
    id          UUID        PRIMARY KEY,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        TEXT        NOT NULL CHECK (kind IN ('email', 'password_reset')),
    code_hash   TEXT        NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ
);

CREATE INDEX verification_codes_user_id_idx ON verification_codes (user_id);

-- Lookup by hash only over unused codes — avoids scanning used rows.
CREATE INDEX verification_codes_hash_active_idx ON verification_codes (code_hash)
    WHERE used_at IS NULL;
