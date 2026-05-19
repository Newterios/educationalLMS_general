CREATE TABLE users (
    id              UUID        PRIMARY KEY,
    email           TEXT        NOT NULL UNIQUE,
    password_hash   TEXT        NOT NULL,
    full_name       TEXT        NOT NULL,
    locale          TEXT        NOT NULL DEFAULT 'en',
    role            TEXT        NOT NULL DEFAULT 'student',
    email_verified  BOOLEAN     NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
