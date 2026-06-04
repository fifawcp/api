DROP INDEX IF EXISTS idx_competitions_one_pool_per_match;
ALTER TABLE competitions DROP CONSTRAINT IF EXISTS chk_competition_pool_match;
ALTER TABLE competitions DROP CONSTRAINT IF EXISTS chk_competition_type;
ALTER TABLE competitions ADD CONSTRAINT chk_competition_type
    CHECK (type IN ('pickem', 'match'));
ALTER TABLE competitions DROP COLUMN IF EXISTS match_id;
