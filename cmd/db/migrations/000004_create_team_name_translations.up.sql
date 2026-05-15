CREATE TABLE IF NOT EXISTS team_name_translations (
  team_fifa_code VARCHAR(8) NOT NULL REFERENCES teams(fifa_code) ON DELETE CASCADE,
  locale         VARCHAR(5) NOT NULL,
  name           VARCHAR(100) NOT NULL,
  PRIMARY KEY (team_fifa_code, locale),
  CHECK (locale IN ('en', 'es'))
);

CREATE INDEX IF NOT EXISTS idx_team_name_translations_locale_team
  ON team_name_translations(locale, team_fifa_code);

CREATE INDEX IF NOT EXISTS idx_team_name_translations_team
  ON team_name_translations(team_fifa_code);

-- Localized read-side projection of teams. Joins teams with their translations
-- and exposes name_translations as a single JSON object keyed by locale,
-- e.g. {"en": "Spain", "es": "España"}.
-- Repositories should select from team_localized instead of teams.
CREATE OR REPLACE VIEW team_localized AS
SELECT
  t.fifa_code,
  t.flag_url,
  t.group_code,
  COALESCE(
    (SELECT json_object_agg(tnt.locale, tnt.name)
     FROM team_name_translations tnt
     WHERE tnt.team_fifa_code = t.fifa_code),
    '{}'::json
  ) AS name_translations
FROM teams t;
