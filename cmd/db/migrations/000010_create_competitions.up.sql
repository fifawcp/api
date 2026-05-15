CREATE TABLE IF NOT EXISTS competitions (
    id         BIGSERIAL PRIMARY KEY,
    board_id   BIGINT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    type       VARCHAR(32) NOT NULL,
    name       VARCHAR(20) NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_competition_type CHECK (type IN ('pickem', 'match')),
    CONSTRAINT competitions_board_id_name_key UNIQUE (board_id, name)
);

CREATE INDEX IF NOT EXISTS idx_competitions_board_id ON competitions(board_id);

-- At most one pickem competition per board.
CREATE UNIQUE INDEX idx_competitions_one_pickem_per_board
    ON competitions(board_id) WHERE type = 'pickem';

-- Stages in scope for a match competition (at least one required).
CREATE TABLE IF NOT EXISTS competition_scope_stages (
    competition_id BIGINT NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    stage          VARCHAR(32) NOT NULL,
    PRIMARY KEY (competition_id, stage),
    CONSTRAINT chk_competition_scope_stage CHECK (stage IN (
        'group_stage', 'round_of_32', 'round_of_16',
        'quarterfinals', 'semifinals', 'third_place', 'final'
    ))
);

-- Optional team filter for match competitions (empty = no filter = all teams).
CREATE TABLE IF NOT EXISTS competition_scope_teams (
    competition_id BIGINT     NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    team_fifa_code VARCHAR(8) NOT NULL REFERENCES teams(fifa_code),
    PRIMARY KEY (competition_id, team_fifa_code)
);

-- Per-pickem-competition leaderboard cache. All hit counts NOT NULL.
CREATE TABLE competition_pickem_scores (
    competition_id        BIGINT NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    user_id               UUID   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_points          INT    NOT NULL DEFAULT 0,
    group_exact_positions INT    NOT NULL DEFAULT 0,
    group_qualifier_hits  INT    NOT NULL DEFAULT 0,
    best_third_hits       INT    NOT NULL DEFAULT 0,
    bracket_hits          INT    NOT NULL DEFAULT 0,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (competition_id, user_id)
);

CREATE INDEX idx_cps_competition_points
    ON competition_pickem_scores(competition_id, total_points DESC);

-- Per-match-competition leaderboard cache. All hit counts NOT NULL.
CREATE TABLE competition_match_scores (
    competition_id   BIGINT NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    user_id          UUID   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_points     INT    NOT NULL DEFAULT 0,
    exact_hits       INT    NOT NULL DEFAULT 0,
    correct_outcomes INT    NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (competition_id, user_id)
);

CREATE INDEX idx_cms_competition_points
    ON competition_match_scores(competition_id, total_points DESC);
