ALTER TABLE competition_pickem_scores DROP COLUMN IF EXISTS award_hits;

ALTER TABLE score_events DROP CONSTRAINT score_events_source_type_check;
ALTER TABLE score_events ADD  CONSTRAINT score_events_source_type_check CHECK (
  source_type IN (
    'group_standing_pick',
    'best_third_pick',
    'bracket_pick',
    'match_score_pick'
  )
);

DROP TABLE IF EXISTS award_winners;
DROP TABLE IF EXISTS user_award_picks;
