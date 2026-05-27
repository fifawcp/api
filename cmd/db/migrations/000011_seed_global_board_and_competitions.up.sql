-- Seed the singleton Global board plus its two default competitions.
-- created_by is NULL because these are system-seeded, not user-created.

INSERT INTO boards (name, privacy, join_code)
VALUES ('Global', 'global', NULL);

WITH global_board AS (
    SELECT id FROM boards WHERE privacy = 'global'
),
pickem AS (
    INSERT INTO competitions (board_id, type, name, created_by)
    SELECT id, 'pickem', 'Pick''em', NULL FROM global_board
    RETURNING id
),
match_competition AS (
    INSERT INTO competitions (board_id, type, name, created_by)
    SELECT id, 'match', 'All Matches', NULL FROM global_board
    RETURNING id
)
INSERT INTO competition_scope_stages (competition_id, stage)
SELECT match_competition.id, stage
FROM match_competition,
     (VALUES
         ('group_stage'), ('round_of_32'), ('round_of_16'),
         ('quarterfinals'), ('semifinals'), ('third_place'), ('final')
     ) AS stages(stage);
