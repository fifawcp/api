CREATE TABLE IF NOT EXISTS team_name_translations (
  team_fifa_code VARCHAR(8) NOT NULL REFERENCES teams(fifa_code) ON DELETE CASCADE,
  locale VARCHAR(5) NOT NULL,
  name VARCHAR(100) NOT NULL,
  PRIMARY KEY (team_fifa_code, locale),
  CHECK (locale IN ('en', 'es'))
);

CREATE INDEX IF NOT EXISTS idx_team_name_translations_locale_team
  ON team_name_translations(locale, team_fifa_code);

CREATE INDEX IF NOT EXISTS idx_team_name_translations_team
  ON team_name_translations(team_fifa_code);

INSERT INTO team_name_translations (team_fifa_code, locale, name)
SELECT fifa_code, 'en', name
FROM teams
ON CONFLICT (team_fifa_code, locale) DO UPDATE
SET name = EXCLUDED.name;

INSERT INTO team_name_translations (team_fifa_code, locale, name) VALUES
('MEX', 'es', 'México'),
('RSA', 'es', 'Sudáfrica'),
('KOR', 'es', 'Corea del Sur'),
('CZE', 'es', 'Chequia'),
('CAN', 'es', 'Canadá'),
('BIH', 'es', 'Bosnia y Herzegovina'),
('QAT', 'es', 'Catar'),
('SUI', 'es', 'Suiza'),
('BRA', 'es', 'Brasil'),
('MAR', 'es', 'Marruecos'),
('HAI', 'es', 'Haití'),
('SCO', 'es', 'Escocia'),
('USA', 'es', 'Estados Unidos'),
('PAR', 'es', 'Paraguay'),
('AUS', 'es', 'Australia'),
('TUR', 'es', 'Turquía'),
('GER', 'es', 'Alemania'),
('CUW', 'es', 'Curazao'),
('CIV', 'es', 'Costa de Marfil'),
('ECU', 'es', 'Ecuador'),
('NED', 'es', 'Países Bajos'),
('JPN', 'es', 'Japón'),
('SWE', 'es', 'Suecia'),
('TUN', 'es', 'Túnez'),
('BEL', 'es', 'Bélgica'),
('EGY', 'es', 'Egipto'),
('IRN', 'es', 'Irán'),
('NZL', 'es', 'Nueva Zelanda'),
('ESP', 'es', 'España'),
('CPV', 'es', 'Cabo Verde'),
('KSA', 'es', 'Arabia Saudita'),
('URU', 'es', 'Uruguay'),
('FRA', 'es', 'Francia'),
('SEN', 'es', 'Senegal'),
('IRQ', 'es', 'Irak'),
('NOR', 'es', 'Noruega'),
('ARG', 'es', 'Argentina'),
('ALG', 'es', 'Argelia'),
('AUT', 'es', 'Austria'),
('JOR', 'es', 'Jordania'),
('POR', 'es', 'Portugal'),
('COD', 'es', 'RD del Congo'),
('UZB', 'es', 'Uzbekistán'),
('COL', 'es', 'Colombia'),
('ENG', 'es', 'Inglaterra'),
('CRO', 'es', 'Croacia'),
('GHA', 'es', 'Ghana'),
('PAN', 'es', 'Panamá')
ON CONFLICT (team_fifa_code, locale) DO UPDATE
SET name = EXCLUDED.name;

ALTER TABLE teams DROP COLUMN IF EXISTS name;

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
