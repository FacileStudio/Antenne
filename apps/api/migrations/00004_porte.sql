-- +goose Up
-- porte owns the credentials from here on: one identity table, one session
-- table, and the login codes that bridge a browser login to a CLI.
--
-- The tables are porte/pg's Schema column for column, with its porte_users
-- left out and every foreign key repointed at Antenne's own users. porte offers
-- UserStore as the escape hatch for exactly this: users is referenced across
-- this codebase and carries the role, which porte transports no opinion about.
-- Do not "tidy" the column names — porte/pg's queries are written against them,
-- so a divergence here surfaces as a runtime error on the login path rather
-- than at boot.

CREATE TABLE IF NOT EXISTS porte_identities (
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    subject         TEXT NOT NULL,
    password_hash   TEXT NOT NULL DEFAULT '',
    access_token    TEXT NOT NULL DEFAULT '',
    refresh_token   TEXT NOT NULL DEFAULT '',
    token_expiry    TIMESTAMPTZ,
    roles           JSONB,
    roles_synced_at TIMESTAMPTZ,
    synced_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (provider, subject)
);
CREATE INDEX IF NOT EXISTS porte_identities_user_idx ON porte_identities (user_id);

CREATE TABLE IF NOT EXISTS porte_sessions (
    id           BIGSERIAL PRIMARY KEY,
    token_hash   TEXT NOT NULL UNIQUE,
    user_id      BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    label        TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS porte_sessions_user_idx ON porte_sessions (user_id);
CREATE INDEX IF NOT EXISTS porte_sessions_expiry_idx ON porte_sessions (expires_at)
    WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS porte_login_codes (
    code_hash   TEXT PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ
);

-- No password is carried across, and that is deliberate rather than an
-- oversight. Journal's adoption copies users.password_hash straight into
-- porte_identities because Journal was already argon2id at porte's own
-- parameters, so the move is byte-identical. Antenne is not: it stored
-- bcrypt over a base64(sha256(password)) prehash, written to survive bcrypt's
-- 72-byte ceiling and its truncation at the first NUL byte. Copied here those
-- bytes are not a valid argon2id encoding, so local.VerifyPassword would refuse
-- them for every input and the login form would answer "invalid email or
-- password" to the correct password, silently.
--
-- Nothing is lost, because ANTENNE_ADMIN_PASSWORD has always been the source of
-- truth for the bootstrap account: the API re-seeds the identity from it at
-- boot, now as argon2id. Changing the variable and restarting is still the
-- recovery path, and it is still the only one.

-- users.password_hash and the old sessions table are both left in place, dead.
-- Dropping them in the same deploy that stops reading them is what makes a
-- rollback impossible for the sake of tidiness: the previous binary needs
-- sessions to exist to authenticate anyone at all. They come out in a later
-- migration, once this one has held.
--
-- The sessions rows are not carried over either. The two tables agree on what
-- they store — the SHA-256 hex of a 32-byte token — so the copy would work, and
-- it would still be pointless: the cookie is named antenne_session and porte
-- reads __Host-session, or `session` where it accepts the unprefixed form.
-- No browser holding one can present it under a name porte looks for, so every
-- session ends at this deploy. With one bootstrap account that is one login.

-- The pre-rename bootstrap account goes. `admin@nook.local` was seeded when this
-- service was still called Nook, and the rename moved the derived address to
-- `admin@antenne.local` without moving the row: since then both ensureAdmin and
-- the login form resolve the new address only, so nothing could reach the old
-- account by password — while Authenticate resolved a session to a user by id
-- and never rechecked the address, so its cookies kept authenticating as ADMIN
-- and survived every rotation of ANTENNE_ADMIN_PASSWORD. An administrator
-- credential with no owner and no revocation path is not something to carry
-- into a new auth system.
--
-- Guarded on having no porte identity so this cannot delete an account somebody
-- has since attached a real credential to, and so a re-run is a no-op.
DELETE FROM users
 WHERE email = 'admin@nook.local'
   AND NOT EXISTS (
       SELECT 1 FROM porte_identities WHERE porte_identities.user_id = users.id
   );

-- +goose Down
DROP TABLE IF EXISTS porte_login_codes;
DROP TABLE IF EXISTS porte_sessions;
DROP TABLE IF EXISTS porte_identities;
