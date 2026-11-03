-- +goose Up
-- Registre owns shared spaces from here on. A local space is linked to the
-- Registre space it mirrors by Registre's id, which survives a rename; NULL
-- means the space was never linked, and Antenne then leaves it alone.
ALTER TABLE spaces ADD COLUMN IF NOT EXISTS registre_space_id TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS spaces_registre_space_id_key ON spaces (registre_space_id);

-- +goose Down
DROP INDEX IF EXISTS spaces_registre_space_id_key;
ALTER TABLE spaces DROP COLUMN IF EXISTS registre_space_id;
