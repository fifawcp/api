ALTER TABLE competitions DROP CONSTRAINT chk_competition_type;
ALTER TABLE competitions DROP CONSTRAINT chk_competition_pick_match;

UPDATE competitions SET type = 'pool' WHERE type = 'pick';

ALTER TABLE competitions ADD CONSTRAINT chk_competition_type
    CHECK (type IN ('pickem', 'match', 'pool', 'awards'));
ALTER TABLE competitions ADD CONSTRAINT chk_competition_pool_match
    CHECK ((type = 'pool') = (match_id IS NOT NULL));

DROP INDEX IF EXISTS idx_competitions_one_pick_per_match;
CREATE UNIQUE INDEX idx_competitions_one_pool_per_match
    ON competitions(board_id, match_id) WHERE type = 'pool';
