-- Seed World Cup 2026 data: teams and initial standings

-- Insert teams (group_code stored directly, no separate groups table)
INSERT INTO teams (fifa_code, name, flag_url, group_code) VALUES
-- Group A
('MEX', 'Mexico', 'https://flagcdn.com/w320/mx.png', 'A'),
('RSA', 'South Africa', 'https://flagcdn.com/w320/za.png', 'A'),
('KOR', 'South Korea', 'https://flagcdn.com/w320/kr.png', 'A'),
('CZE', 'Czechia', 'https://flagcdn.com/w320/cz.png', 'A'),

-- Group B
('CAN', 'Canada', 'https://flagcdn.com/w320/ca.png', 'B'),
('BIH', 'Bosnia and Herzegovina', 'https://flagcdn.com/w320/ba.png', 'B'),
('QAT', 'Qatar', 'https://flagcdn.com/w320/qa.png', 'B'),
('SUI', 'Switzerland', 'https://flagcdn.com/w320/ch.png', 'B'),

-- Group C
('BRA', 'Brazil', 'https://flagcdn.com/w320/br.png', 'C'),
('MAR', 'Morocco', 'https://flagcdn.com/w320/ma.png', 'C'),
('HAI', 'Haiti', 'https://flagcdn.com/w320/ht.png', 'C'),
('SCO', 'Scotland', 'https://flagcdn.com/w320/gb-sct.png', 'C'),

-- Group D
('USA', 'United States', 'https://flagcdn.com/w320/us.png', 'D'),
('PAR', 'Paraguay', 'https://flagcdn.com/w320/py.png', 'D'),
('AUS', 'Australia', 'https://flagcdn.com/w320/au.png', 'D'),
('TUR', 'Türkiye', 'https://flagcdn.com/w320/tr.png', 'D'),

-- Group E
('GER', 'Germany', 'https://flagcdn.com/w320/de.png', 'E'),
('CUW', 'Curaçao', 'https://flagcdn.com/w320/cw.png', 'E'),
('CIV', 'Ivory Coast', 'https://flagcdn.com/w320/ci.png', 'E'),
('ECU', 'Ecuador', 'https://flagcdn.com/w320/ec.png', 'E'),

-- Group F
('NED', 'Netherlands', 'https://flagcdn.com/w320/nl.png', 'F'),
('JPN', 'Japan', 'https://flagcdn.com/w320/jp.png', 'F'),
('SWE', 'Sweden', 'https://flagcdn.com/w320/se.png', 'F'),
('TUN', 'Tunisia', 'https://flagcdn.com/w320/tn.png', 'F'),

-- Group G
('BEL', 'Belgium', 'https://flagcdn.com/w320/be.png', 'G'),
('EGY', 'Egypt', 'https://flagcdn.com/w320/eg.png', 'G'),
('IRN', 'Iran', 'https://flagcdn.com/w320/ir.png', 'G'),
('NZL', 'New Zealand', 'https://flagcdn.com/w320/nz.png', 'G'),

-- Group H
('ESP', 'Spain', 'https://flagcdn.com/w320/es.png', 'H'),
('CPV', 'Cape Verde', 'https://flagcdn.com/w320/cv.png', 'H'),
('KSA', 'Saudi Arabia', 'https://flagcdn.com/w320/sa.png', 'H'),
('URU', 'Uruguay', 'https://flagcdn.com/w320/uy.png', 'H'),

-- Group I
('FRA', 'France', 'https://flagcdn.com/w320/fr.png', 'I'),
('SEN', 'Senegal', 'https://flagcdn.com/w320/sn.png', 'I'),
('IRQ', 'Iraq', 'https://flagcdn.com/w320/iq.png', 'I'),
('NOR', 'Norway', 'https://flagcdn.com/w320/no.png', 'I'),

-- Group J
('ARG', 'Argentina', 'https://flagcdn.com/w320/ar.png', 'J'),
('ALG', 'Algeria', 'https://flagcdn.com/w320/dz.png', 'J'),
('AUT', 'Austria', 'https://flagcdn.com/w320/at.png', 'J'),
('JOR', 'Jordan', 'https://flagcdn.com/w320/jo.png', 'J'),

-- Group K
('POR', 'Portugal', 'https://flagcdn.com/w320/pt.png', 'K'),
('COD', 'DR Congo', 'https://flagcdn.com/w320/cd.png', 'K'),
('UZB', 'Uzbekistan', 'https://flagcdn.com/w320/uz.png', 'K'),
('COL', 'Colombia', 'https://flagcdn.com/w320/co.png', 'K'),

-- Group L
('ENG', 'England', 'https://flagcdn.com/w320/gb-eng.png', 'L'),
('CRO', 'Croatia', 'https://flagcdn.com/w320/hr.png', 'L'),
('GHA', 'Ghana', 'https://flagcdn.com/w320/gh.png', 'L'),
('PAN', 'Panama', 'https://flagcdn.com/w320/pa.png', 'L');

-- Insert initial group standings (fifa_code is PK, group_code stored directly)
INSERT INTO group_standings (fifa_code, group_code, position, matches_played, wins, draws, losses, goals_for, goals_against, goal_difference, points)
SELECT
  t.fifa_code,
  t.group_code,
  0 as position,
  0 as matches_played,
  0 as wins,
  0 as draws,
  0 as losses,
  0 as goals_for,
  0 as goals_against,
  0 as goal_difference,
  0 as points
FROM teams t;
