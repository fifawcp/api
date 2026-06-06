-- Awards becomes a standalone competition type. Pick'em and Awards leaderboards
-- are computed on read from score_events, so their per-competition cache tables
-- are removed (competition_pickem_scores dropped here; awards never cached).
-- Match/pool keep competition_match_scores.

ALTER TABLE competitions DROP CONSTRAINT chk_competition_type;
ALTER TABLE competitions ADD CONSTRAINT chk_competition_type
    CHECK (type IN ('pickem', 'match', 'pool', 'awards'));

CREATE UNIQUE INDEX idx_competitions_one_awards_per_board
    ON competitions(board_id) WHERE type = 'awards';

INSERT INTO competitions (board_id, type, name, created_by, created_at)
SELECT id, 'awards', 'Awards', NULL, NOW() FROM boards
ON CONFLICT (board_id, name) DO NOTHING;

DROP TABLE competition_pickem_scores;
