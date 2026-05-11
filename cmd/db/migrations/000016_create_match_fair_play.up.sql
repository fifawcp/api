CREATE TABLE match_fair_play (
    match_id                BIGINT     NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    team_fifa_code          VARCHAR(8) NOT NULL REFERENCES teams(fifa_code),
    yellow_cards            SMALLINT   NOT NULL DEFAULT 0,
    indirect_red_cards      SMALLINT   NOT NULL DEFAULT 0,
    direct_red_cards        SMALLINT   NOT NULL DEFAULT 0,
    yellow_direct_red_cards SMALLINT   NOT NULL DEFAULT 0,
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (match_id, team_fifa_code)
);
