-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT        NOT NULL UNIQUE,
    name          TEXT        NOT NULL DEFAULT '',
    password_hash TEXT        NOT NULL DEFAULT '',
    role          TEXT        NOT NULL DEFAULT 'ADMIN',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT        PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS app_settings (
    id           INTEGER     PRIMARY KEY DEFAULT 1,
    timeout      INTEGER     NOT NULL DEFAULT 10,
    listen_port  INTEGER     NOT NULL DEFAULT 9090,
    system       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    pool         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT app_settings_singleton CHECK (id = 1)
);

CREATE TABLE IF NOT EXISTS providers (
    id         TEXT        PRIMARY KEY,
    type       TEXT        NOT NULL,
    name       TEXT        NOT NULL DEFAULT '',
    enabled    BOOLEAN     NOT NULL DEFAULT true,
    position   INTEGER     NOT NULL DEFAULT 0,
    tags       JSONB       NOT NULL DEFAULT '[]'::jsonb,
    config     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_providers_position ON providers (position);

CREATE TABLE IF NOT EXISTS delivery_targets (
    id            TEXT        PRIMARY KEY,
    type          TEXT        NOT NULL,
    name          TEXT        NOT NULL DEFAULT '',
    enabled       BOOLEAN     NOT NULL DEFAULT true,
    position      INTEGER     NOT NULL DEFAULT 0,
    provider_ids  JSONB       NOT NULL DEFAULT '[]'::jsonb,
    provider_tags JSONB       NOT NULL DEFAULT '[]'::jsonb,
    config        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_delivery_targets_position ON delivery_targets (position);

CREATE TABLE IF NOT EXISTS events (
    id            TEXT        PRIMARY KEY,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    source        TEXT        NOT NULL,
    provider_id   TEXT,
    provider_name TEXT        NOT NULL DEFAULT '',
    provider_type TEXT        NOT NULL DEFAULT '',
    event         TEXT        NOT NULL DEFAULT '',
    message       TEXT,
    envelope      JSONB,
    deliveries    JSONB       NOT NULL DEFAULT '[]'::jsonb,
    search_text   TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_provider ON events (provider_id);
CREATE INDEX IF NOT EXISTS idx_events_source ON events (source);

CREATE TABLE IF NOT EXISTS spaces (
    id          TEXT        PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS space_members (
    space_id  TEXT        NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    user_id   TEXT        NOT NULL,
    role      TEXT        NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (space_id, user_id)
);

CREATE TABLE IF NOT EXISTS pool_apps (
    app_id          TEXT        PRIMARY KEY,
    app             TEXT        NOT NULL UNIQUE,
    token           TEXT        NOT NULL UNIQUE,
    emit_channels   JSONB       NOT NULL DEFAULT '[]'::jsonb,
    listen_channels JSONB       NOT NULL DEFAULT '[]'::jsonb,
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pool_apps_token ON pool_apps (token);

CREATE TABLE IF NOT EXISTS pool_channel_offsets (
    channel     TEXT   PRIMARY KEY,
    next_offset BIGINT NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS pool_events (
    id         TEXT        PRIMARY KEY,
    channel    TEXT        NOT NULL,
    offset_num BIGINT      NOT NULL,
    payload    JSONB       NOT NULL,
    sender     TEXT        NOT NULL,
    emitted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_events_channel_offset ON pool_events (channel, offset_num);
CREATE INDEX IF NOT EXISTS idx_pool_events_emitted_at ON pool_events (emitted_at);

CREATE TABLE IF NOT EXISTS pool_subscriptions (
    subscriber_id TEXT   NOT NULL,
    channel       TEXT   NOT NULL,
    last_acked    BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (subscriber_id, channel)
);

-- +goose Down
DROP TABLE IF EXISTS pool_subscriptions;
DROP TABLE IF EXISTS pool_events;
DROP TABLE IF EXISTS pool_channel_offsets;
DROP TABLE IF EXISTS pool_apps;
DROP TABLE IF EXISTS space_members;
DROP TABLE IF EXISTS spaces;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS delivery_targets;
DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS app_settings;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
