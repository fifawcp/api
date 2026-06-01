-- Give every player a single, consistent, JS-safe primary key.
--
-- Players were seeded from two sources: ~879 with their real API-Football IDs
-- (small, e.g. Messi = 45843) and ~471 with random 18-19 digit int64 placeholder
-- IDs. The placeholders exceed JavaScript's Number.MAX_SAFE_INTEGER
-- (9007199254740991, ~2^53), so the web client silently rounds them on
-- JSON.parse and sends a corrupted player_id back, breaking award picks
-- (PUT /api/awards -> 404). The two ID spaces are also just inconsistent.
--
-- players.id is a purely internal surrogate key (nothing syncs players from
-- API-Football by it), so we renumber ALL players into one uniform sequential
-- range and move the real API-Football ID to a separate provider_id column
-- (NULL for the placeholders). This fixes the bug, makes IDs consistent, and
-- preserves the real provider IDs for future use (e.g. resolving award winners
-- from API-Football player stats).

ALTER TABLE players ADD COLUMN provider_id BIGINT;

-- Preserve the real API-Football IDs (the small ones); the random placeholders
-- carry no meaning and become NULL.
UPDATE players SET provider_id = id WHERE id <= 9007199254740991;

-- Uniform sequential PKs for everyone, ordered deterministically so the result
-- is identical on every database (fresh dev, test, and prod).
CREATE TEMP TABLE id_remap ON COMMIT DROP AS
  SELECT id AS old_id,
         row_number() OVER (ORDER BY team_fifa_code, name, id) AS new_id
  FROM players;

-- Updating a referenced primary key requires dropping the FK constraints, since
-- they are not ON UPDATE CASCADE. Drop, remap parent + children, then re-add.
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
