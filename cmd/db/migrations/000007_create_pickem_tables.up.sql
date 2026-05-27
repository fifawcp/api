CREATE TABLE IF NOT EXISTS user_group_picks (
  user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  team_fifa_code     VARCHAR(8) NOT NULL REFERENCES teams(fifa_code),
  team_group_code    CHAR(1) NOT NULL CHECK (
    team_group_code IN ('A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L')
  ),
  predicted_position SMALLINT NOT NULL CHECK (predicted_position BETWEEN 1 AND 4),
  PRIMARY KEY (user_id, team_fifa_code),
  CONSTRAINT uq_user_group_position UNIQUE (user_id, team_group_code, predicted_position)
);

CREATE INDEX IF NOT EXISTS idx_user_group_picks_user_id         ON user_group_picks(user_id);
CREATE INDEX IF NOT EXISTS idx_user_group_picks_team_group_code ON user_group_picks(team_group_code);

CREATE TABLE IF NOT EXISTS user_group_locks (
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  group_code CHAR(1) NOT NULL CHECK (
    group_code IN ('A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L')
  ),
  PRIMARY KEY (user_id, group_code)
);

CREATE INDEX IF NOT EXISTS idx_user_group_locks_user_id ON user_group_locks(user_id);

CREATE TABLE IF NOT EXISTS user_best_third_picks (
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  team_fifa_code VARCHAR(8) NOT NULL REFERENCES teams(fifa_code),
  PRIMARY KEY (user_id, team_fifa_code)
);

CREATE INDEX IF NOT EXISTS idx_user_best_third_picks_user_id ON user_best_third_picks(user_id);

CREATE TABLE IF NOT EXISTS user_bracket_picks (
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  match_id       BIGINT NOT NULL REFERENCES matches(id),
  team_fifa_code VARCHAR(8) NOT NULL REFERENCES teams(fifa_code),
  PRIMARY KEY (user_id, match_id)
);

CREATE INDEX IF NOT EXISTS idx_user_bracket_picks_user_id  ON user_bracket_picks(user_id);
CREATE INDEX IF NOT EXISTS idx_user_bracket_picks_match_id ON user_bracket_picks(match_id);

CREATE TABLE IF NOT EXISTS user_match_score_picks (
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  match_id   BIGINT NOT NULL REFERENCES matches(id),
  home_score SMALLINT NOT NULL CHECK (home_score BETWEEN 0 AND 20),
  away_score SMALLINT NOT NULL CHECK (away_score BETWEEN 0 AND 20),
  PRIMARY KEY (user_id, match_id)
);

CREATE INDEX IF NOT EXISTS idx_user_match_score_picks_user_id  ON user_match_score_picks(user_id);
CREATE INDEX IF NOT EXISTS idx_user_match_score_picks_match_id ON user_match_score_picks(match_id);

CREATE TABLE IF NOT EXISTS score_events (
  id          BIGSERIAL PRIMARY KEY,
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  source_type VARCHAR(32) NOT NULL CHECK (
    source_type IN ('group_standing_pick', 'best_third_pick', 'bracket_pick', 'match_score_pick')
  ),
  source_ref  VARCHAR(64) NOT NULL,
  points      INT NOT NULL,
  created_at  TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_score_event UNIQUE (user_id, source_type, source_ref)
);

CREATE INDEX IF NOT EXISTS idx_score_events_user_id ON score_events(user_id);
