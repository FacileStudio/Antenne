-- +goose Up
-- Health lives in its own table, not in columns on delivery_targets.
-- settings.Store keeps the whole configuration document in memory and rewrites
-- delivery_targets.config from json.Marshal(target) on every dashboard save, so
-- anything hung off that struct is clobbered by a stale in-memory copy the next
-- time an operator touches an unrelated field.
--
-- The foreign key is load-bearing rather than decorative: settings.Store deletes
-- removed targets with DELETE ... WHERE id NOT IN (...), and without the cascade
-- every deleted target would leave an orphan health row that the dashboard would
-- happily report on.
CREATE TABLE IF NOT EXISTS delivery_target_health (
    target_id             TEXT        PRIMARY KEY REFERENCES delivery_targets (id) ON DELETE CASCADE,
    state                 TEXT        NOT NULL DEFAULT 'unknown',
    consecutive_failures  INTEGER     NOT NULL DEFAULT 0,
    consecutive_successes INTEGER     NOT NULL DEFAULT 0,
    last_observed_at      TIMESTAMPTZ,
    last_success_at       TIMESTAMPTZ,
    last_failure_at       TIMESTAMPTZ,
    last_source           TEXT        NOT NULL DEFAULT '',
    last_kind             TEXT        NOT NULL DEFAULT '',
    last_detail           TEXT        NOT NULL DEFAULT '',
    down_since            TIMESTAMPTZ,
    last_alert_at         TIMESTAMPTZ,
    alert_count           INTEGER     NOT NULL DEFAULT 0,
    transitions           JSONB       NOT NULL DEFAULT '[]'::jsonb,
    probe_suspended_until TIMESTAMPTZ,
    probe_backoff_level   INTEGER     NOT NULL DEFAULT 0,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- No secondary indexes on purpose. There is one row per delivery target and a
-- self-hosted instance has a handful of them, so every read here is a sequential
-- scan of a page or two; an index on a low-cardinality state column would cost
-- writes on every observation and buy nothing.

-- +goose Down
DROP TABLE IF EXISTS delivery_target_health;
