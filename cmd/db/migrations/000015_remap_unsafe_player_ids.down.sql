-- The id renumbering is irreversible (original placeholder IDs were meaningless);
-- only the provider_id column is dropped.
ALTER TABLE players DROP COLUMN IF EXISTS provider_id;
