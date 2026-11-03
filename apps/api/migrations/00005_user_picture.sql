-- +goose Up
-- The picture is the identity provider's stable URL, never a copy of its bytes.
-- Registre serves the current image behind it, so storing it once is enough for
-- a change to show on the next page load. It is backfilled at boot from
-- porte_identities, because the issuer it is built on is configuration and has
-- no business in a migration.
ALTER TABLE users ADD COLUMN IF NOT EXISTS picture_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS picture_url;
