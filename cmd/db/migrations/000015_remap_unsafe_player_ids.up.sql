-- ~471 players were seeded with random 18-19 digit placeholder IDs that exceed
-- Number.MAX_SAFE_INTEGER, so the web client rounds them on JSON.parse and breaks
-- award picks. players.id is a purely internal surrogate key, so renumber every
-- player into one uniform sequential range and keep the real API-Football ID
-- (where known) in a separate provider_id column.

ALTER TABLE players ADD COLUMN provider_id BIGINT;
UPDATE players SET provider_id = id WHERE id <= 9007199254740991;

-- Ordered deterministically so the result is identical on every database.
CREATE TEMP TABLE id_remap ON COMMIT DROP AS
  SELECT id AS old_id,
         row_number() OVER (ORDER BY team_fifa_code, name, id) AS new_id
  FROM players;

-- The FKs aren't ON UPDATE CASCADE, so drop them to remap the PK, then re-add.
ALTER TABLE user_award_picks DROP CONSTRAINT user_award_picks_player_id_fkey;
ALTER TABLE award_winners    DROP CONSTRAINT award_winners_player_id_fkey;

UPDATE players p          SET id        = r.new_id FROM id_remap r WHERE p.id = r.old_id;
UPDATE user_award_picks u SET player_id = r.new_id FROM id_remap r WHERE u.player_id = r.old_id;
UPDATE award_winners a    SET player_id = r.new_id FROM id_remap r WHERE a.player_id = r.old_id;

ALTER TABLE user_award_picks
  ADD CONSTRAINT user_award_picks_player_id_fkey
  FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE award_winners
  ADD CONSTRAINT award_winners_player_id_fkey
  FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE RESTRICT;
