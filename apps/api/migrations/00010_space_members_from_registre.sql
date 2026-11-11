-- +goose Up
-- Marks a membership the Registre import wrote. The import no longer creates an
-- account for a member who never signed in, so such a membership has no user row
-- to tell it apart from a local one, and a sync must still remove it once
-- Registre drops the member. Existing rows keep false: the ones the import wrote
-- before this belong to pre-provisioned users with a Registre identity, which the
-- sync already removes by that identity.
ALTER TABLE space_members ADD COLUMN IF NOT EXISTS from_registre BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE space_members DROP COLUMN IF EXISTS from_registre;
