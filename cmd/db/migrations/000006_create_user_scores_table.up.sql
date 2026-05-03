CREATE TABLE IF NOT EXISTS user_scores (
  user_id            UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  total_points       INT NOT NULL DEFAULT 0,
  pickem_points      INT NOT NULL DEFAULT 0,
  match_score_points INT NOT NULL DEFAULT 0,
  exact_hits         INT NOT NULL DEFAULT 0,
  correct_outcomes   INT NOT NULL DEFAULT 0,
  updated_at         TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW()
);
