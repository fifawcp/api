DROP VIEW IF EXISTS team_localized;

ALTER TABLE teams ADD COLUMN IF NOT EXISTS name VARCHAR(100);

UPDATE teams t
SET name = COALESCE(en_t.name, es_t.name)
FROM team_name_translations en_t
LEFT JOIN team_name_translations es_t
  ON es_t.team_fifa_code = en_t.team_fifa_code
  AND es_t.locale = 'es'
WHERE en_t.team_fifa_code = t.fifa_code
  AND en_t.locale = 'en';

-- Fill any remaining nulls from Spanish-only records.
UPDATE teams t
SET name = es_t.name
FROM team_name_translations es_t
WHERE es_t.team_fifa_code = t.fifa_code
  AND es_t.locale = 'es'
  AND t.name IS NULL;

ALTER TABLE teams ALTER COLUMN name SET NOT NULL;

DROP INDEX IF EXISTS idx_team_name_translations_team;
DROP INDEX IF EXISTS idx_team_name_translations_locale_team;
DROP TABLE IF EXISTS team_name_translations;
