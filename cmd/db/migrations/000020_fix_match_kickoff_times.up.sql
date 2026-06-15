-- ARG vs ALG: was 2026-06-16 (a day early); real fixture is Jun 16, 21:00 ET.
UPDATE matches SET kickoff_at = '2026-06-17T01:00:00Z'
  WHERE stage_code = 'group_stage' AND home_team_fifa_code = 'ARG' AND away_team_fifa_code = 'ALG';

-- COL vs COD: was 18:00 ET; real fixture is Jun 23, 22:00 ET.
UPDATE matches SET kickoff_at = '2026-06-24T02:00:00Z'
  WHERE stage_code = 'group_stage' AND home_team_fifa_code = 'COL' AND away_team_fifa_code = 'COD';
