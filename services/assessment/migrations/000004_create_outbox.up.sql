CREATE TABLE IF NOT EXISTS outbox (
    id           BIGSERIAL   PRIMARY KEY,
    aggregate_id UUID        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      BYTEA       NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;
