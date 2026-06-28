-- Knockout kickoff times were seeded in strict chronological order against match
-- IDs 73-104, but FIFA's match numbering is not chronological within a day, so the
-- correct per-day times landed on the wrong matches. Slot rules and venues are keyed
-- to match ID and stayed correct; only kickoff_at needs reassigning. Keyed by match id
-- because knockout teams are NULL until groups resolve. Comments show FIFA local time.

-- Round of 32
UPDATE matches SET kickoff_at = '2026-06-29T20:30:00Z' WHERE id = 74; -- 1E v 3rd, Gillette, Foxborough — Jun 29, 4:30 PM ET
UPDATE matches SET kickoff_at = '2026-06-30T01:00:00Z' WHERE id = 75; -- 1F v 2C, Estadio BBVA, Guadalupe — Jun 29, 7:00 PM CST
UPDATE matches SET kickoff_at = '2026-06-29T17:00:00Z' WHERE id = 76; -- 1C v 2F, NRG, Houston — Jun 29, 12:00 PM CT
UPDATE matches SET kickoff_at = '2026-06-30T21:00:00Z' WHERE id = 77; -- 1I v 3rd, MetLife, East Rutherford — Jun 30, 5:00 PM ET
UPDATE matches SET kickoff_at = '2026-06-30T17:00:00Z' WHERE id = 78; -- 2E v 2I, AT&T, Arlington — Jun 30, 12:00 PM CT
UPDATE matches SET kickoff_at = '2026-07-02T00:00:00Z' WHERE id = 81; -- 1D v 3rd, Levi's, Santa Clara — Jul 1, 5:00 PM PT
UPDATE matches SET kickoff_at = '2026-07-01T20:00:00Z' WHERE id = 82; -- 1G v 3rd, Lumen, Seattle — Jul 1, 1:00 PM PT
UPDATE matches SET kickoff_at = '2026-07-02T23:00:00Z' WHERE id = 83; -- 2K v 2L, BMO, Toronto — Jul 2, 7:00 PM ET
UPDATE matches SET kickoff_at = '2026-07-02T19:00:00Z' WHERE id = 84; -- 1H v 2J, SoFi, Inglewood — Jul 2, 12:00 PM PT
UPDATE matches SET kickoff_at = '2026-07-03T22:00:00Z' WHERE id = 86; -- 1J v 2H, Hard Rock, Miami — Jul 3, 6:00 PM ET
UPDATE matches SET kickoff_at = '2026-07-04T01:30:00Z' WHERE id = 87; -- 1K v 3rd, Arrowhead, Kansas City — Jul 3, 8:30 PM CT
UPDATE matches SET kickoff_at = '2026-07-03T18:00:00Z' WHERE id = 88; -- 2D v 2G, AT&T, Arlington — Jul 3, 1:00 PM CT

-- Round of 16 (Jul 4 swap)
UPDATE matches SET kickoff_at = '2026-07-04T21:00:00Z' WHERE id = 89; -- W74 v W77, Lincoln Financial, Philadelphia — Jul 4, 5:00 PM ET
UPDATE matches SET kickoff_at = '2026-07-04T17:00:00Z' WHERE id = 90; -- W73 v W75, NRG, Houston — Jul 4, 12:00 PM CT
