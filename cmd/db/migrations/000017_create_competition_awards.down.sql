-- Recreates competition_pickem_scores empty (it's a cache — repopulate with a
-- rescore). Restores the pre-000017 schema: 000010 columns + the 000013 award_hits.

CREATE TABLE competition_pickem_scores (
    competition_id        BIGINT NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    user_id               UUID   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_points          INT    NOT NULL DEFAULT 0,
    group_exact_positions INT    NOT NULL DEFAULT 0,
    group_qualifier_hits  INT    NOT NULL DEFAULT 0,
    best_third_hits       INT    NOT NULL DEFAULT 0,
    bracket_hits          INT    NOT NULL DEFAULT 0,
    award_hits            INT    NOT NULL DEFAULT 0,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (competition_id, user_id)
);

CREATE INDEX idx_cps_competition_points
    ON competition_pickem_scores(competition_id, total_points DESC);

DELETE FROM competitions WHERE type = 'awards';
DROP INDEX IF EXISTS idx_competitions_one_awards_per_board;

ALTER TABLE competitions DROP CONSTRAINT chk_competition_type;
ALTER TABLE competitions ADD CONSTRAINT chk_competition_type
    CHECK (type IN ('pickem', 'match', 'pool'));
