-- +goose Up
-- The keyed hash of the pool secret a token was issued under. A token is
-- accepted only while this matches the current secret, so rotating the secret
-- revokes every token issued under the old one. The secret itself is never
-- stored here. Existing rows start NULL and the server stamps them with the
-- current secret's fingerprint at boot, so nothing disconnects at deploy.
ALTER TABLE pool_apps ADD COLUMN IF NOT EXISTS secret_fingerprint TEXT;

-- +goose Down
ALTER TABLE pool_apps DROP COLUMN IF EXISTS secret_fingerprint;
