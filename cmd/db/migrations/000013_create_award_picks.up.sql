CREATE TABLE IF NOT EXISTS user_award_picks (
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  award_type VARCHAR(16) NOT NULL CHECK (
    award_type IN ('golden_boot', 'golden_ball', 'golden_glove', 'young_player')
  ),
  player_id  BIGINT      NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, award_type)
);

CREATE INDEX IF NOT EXISTS idx_user_award_picks_user_id         ON user_award_picks(user_id);
CREATE INDEX IF NOT EXISTS idx_user_award_picks_award_player    ON user_award_picks(award_type, player_id);

CREATE TABLE IF NOT EXISTS award_winners (
  award_type VARCHAR(16) PRIMARY KEY CHECK (
    award_type IN ('golden_boot', 'golden_ball', 'golden_glove', 'young_player')
  ),
  player_id  BIGINT      NOT NULL REFERENCES players(id) ON DELETE RESTRICT,
  updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Extend the score_events source_type check to include award picks
ALTER TABLE score_events DROP CONSTRAINT score_events_source_type_check;
ALTER TABLE score_events ADD  CONSTRAINT score_events_source_type_check CHECK (
  source_type IN (
    'group_standing_pick',
    'best_third_pick',
    'bracket_pick',
    'match_score_pick',
    'award_pick'
  )
);

ALTER TABLE competition_pickem_scores ADD COLUMN IF NOT EXISTS award_hits INT NOT NULL DEFAULT 0;
