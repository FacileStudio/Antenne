-- +goose Up
-- porte v0.10 binds a CLI or mobile login code to the identity that vouched for
-- it. Without these two columns the CLI login fails.
ALTER TABLE porte_login_codes ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT '';
ALTER TABLE porte_login_codes ADD COLUMN IF NOT EXISTS subject TEXT NOT NULL DEFAULT '';

-- The person's language, from the OIDC locale claim. porte serves no /auth/me,
-- so the dashboard reads it from the session endpoint.
ALTER TABLE users ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'en';

-- One row holding the GDPR deletion receiver's cursor and handled subjects.
CREATE TABLE IF NOT EXISTS deletion_state (
    name       TEXT PRIMARY KEY,
    state      BYTEA       NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS deletion_state;
ALTER TABLE users DROP COLUMN IF EXISTS locale;
ALTER TABLE porte_login_codes DROP COLUMN IF EXISTS subject;
ALTER TABLE porte_login_codes DROP COLUMN IF EXISTS provider;
