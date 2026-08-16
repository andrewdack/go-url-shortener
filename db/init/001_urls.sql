CREATE TABLE IF NOT EXISTS urls (
    short_url  TEXT PRIMARY KEY,
    long_url   TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    clicks     BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
