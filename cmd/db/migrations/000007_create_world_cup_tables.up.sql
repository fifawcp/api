CREATE TABLE IF NOT EXISTS teams (
  fifa_code VARCHAR(8) PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  flag_url TEXT,
  group_code CHAR(1) NOT NULL CHECK (
    group_code IN ('A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L')
  )
);

CREATE INDEX IF NOT EXISTS idx_teams_group_code ON teams(group_code);

CREATE TABLE IF NOT EXISTS matches (
  id BIGSERIAL PRIMARY KEY,
  stage_code VARCHAR(32) NOT NULL,
  group_code CHAR(1),
  home_team_fifa_code VARCHAR(8) REFERENCES teams(fifa_code),
  away_team_fifa_code VARCHAR(8) REFERENCES teams(fifa_code),
  kickoff_at TIMESTAMP(0) WITH TIME ZONE NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
  home_score SMALLINT,
  away_score SMALLINT,
  winner_team_fifa_code VARCHAR(8) REFERENCES teams(fifa_code),
  updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  CHECK (stage_code IN (
    'group_stage',
    'round_of_32',
    'round_of_16',
    'quarterfinals',
    'semifinals',
    'third_place',
    'final'
  )),
  CHECK (status IN ('scheduled', 'finished')),
  CHECK (group_code IS NULL OR group_code IN ('A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L')),
  CONSTRAINT check_winner_is_home_or_away 
  CHECK (
    winner_team_fifa_code IS NULL OR 
    (home_team_fifa_code IS NOT NULL AND 
      away_team_fifa_code IS NOT NULL AND 
      (winner_team_fifa_code = home_team_fifa_code OR 
        winner_team_fifa_code = away_team_fifa_code))
  )
);

CREATE INDEX IF NOT EXISTS idx_matches_stage_code ON matches(stage_code);
CREATE INDEX IF NOT EXISTS idx_matches_group_code ON matches(group_code);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_matches_kickoff_at ON matches(kickoff_at);
CREATE INDEX IF NOT EXISTS idx_matches_home_team_fifa_code ON matches(home_team_fifa_code);
CREATE INDEX IF NOT EXISTS idx_matches_away_team_fifa_code ON matches(away_team_fifa_code);

-- Materialized table for group standings
-- Updated after each group stage match finishes
-- Sorted by FIFA tiebreakers: points → goal_difference → goals_for → head-to-head
CREATE TABLE IF NOT EXISTS group_standings (
  fifa_code VARCHAR(8) PRIMARY KEY REFERENCES teams(fifa_code) ON DELETE CASCADE,
  group_code CHAR(1) NOT NULL CHECK (group_code IN ('A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L')),
  position SMALLINT NOT NULL,
  matches_played SMALLINT NOT NULL DEFAULT 0,
  wins SMALLINT NOT NULL DEFAULT 0,
  draws SMALLINT NOT NULL DEFAULT 0,
  losses SMALLINT NOT NULL DEFAULT 0,
  goals_for SMALLINT NOT NULL DEFAULT 0,
  goals_against SMALLINT NOT NULL DEFAULT 0,
  goal_difference SMALLINT NOT NULL DEFAULT 0,
  points SMALLINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_group_standings_group_code ON group_standings(group_code);
CREATE INDEX IF NOT EXISTS idx_group_standings_position ON group_standings(group_code, position);
