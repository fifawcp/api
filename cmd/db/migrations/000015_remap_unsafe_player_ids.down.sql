-- Drops the provider_id column. The uniform sequential id renumbering is not
-- reversible (the original random placeholder IDs were meaningless and are not
-- reconstructable), and the new IDs are perfectly valid, so player.id values are
-- intentionally left in place.
ALTER TABLE players DROP COLUMN IF EXISTS provider_id;
