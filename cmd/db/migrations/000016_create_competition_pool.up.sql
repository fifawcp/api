ALTER TABLE competitions ADD COLUMN match_id BIGINT REFERENCES matches(id) ON DELETE CASCADE;

-- Postgres cannot edit a CHECK in place; drop and re-add to allow the new type.
ALTER TABLE competitions DROP CONSTRAINT chk_competition_type;
ALTER TABLE competitions ADD CONSTRAINT chk_competition_type
    CHECK (type IN ('pickem', 'match', 'pool'));

-- match_id is present iff the competition is a pool.
ALTER TABLE competitions ADD CONSTRAINT chk_competition_pool_match
    CHECK ((type = 'pool') = (match_id IS NOT NULL));

-- At most one pool per match per board.
CREATE UNIQUE INDEX idx_competitions_one_pool_per_match
    ON competitions(board_id, match_id) WHERE type = 'pool';
