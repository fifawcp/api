DELETE FROM boards WHERE privacy = 'public';

ALTER TABLE boards DROP CONSTRAINT chk_boards_privacy_shape;
ALTER TABLE boards DROP COLUMN privacy;

DROP INDEX boards_join_code_unique;

ALTER TABLE boards ADD CONSTRAINT boards_join_code_key UNIQUE (join_code);
ALTER TABLE boards ALTER COLUMN owner_user_id SET NOT NULL;
ALTER TABLE boards ALTER COLUMN join_code     SET NOT NULL;
