-- The single-match competition type is renamed 'pool' -> 'pick': it covers ONE
-- match, not many. Both CHECK constraints reject the new value, so drop them
-- first, rename the existing rows, then re-add the constraints and the index.
ALTER TABLE competitions DROP CONSTRAINT chk_competition_type;
ALTER TABLE competitions DROP CONSTRAINT chk_competition_pool_match;

UPDATE competitions SET type = 'pick' WHERE type = 'pool';

ALTER TABLE competitions ADD CONSTRAINT chk_competition_type
    CHECK (type IN ('pickem', 'match', 'pick', 'awards'));
ALTER TABLE competitions ADD CONSTRAINT chk_competition_pick_match
    CHECK ((type = 'pick') = (match_id IS NOT NULL));

DROP INDEX IF EXISTS idx_competitions_one_pool_per_match;
CREATE UNIQUE INDEX idx_competitions_one_pick_per_match
    ON competitions(board_id, match_id) WHERE type = 'pick';
