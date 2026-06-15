-- UZB vs COL: was 18:00 ET; real fixture is Jun 17, 22:00 ET.
UPDATE matches SET kickoff_at = '2026-06-18T02:00:00Z'
  WHERE stage_code = 'group_stage' AND home_team_fifa_code = 'UZB' AND away_team_fifa_code = 'COL';
