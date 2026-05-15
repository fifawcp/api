-- Cascade deletes from boards to competitions, scope tables, and scores.
DELETE FROM boards WHERE privacy = 'global';
