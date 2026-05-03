ALTER TABLE boards ALTER COLUMN owner_user_id DROP NOT NULL;
ALTER TABLE boards ALTER COLUMN join_code     DROP NOT NULL;

ALTER TABLE boards DROP CONSTRAINT boards_join_code_key;
CREATE UNIQUE INDEX boards_join_code_unique ON boards(join_code) WHERE join_code IS NOT NULL;

ALTER TABLE boards
  ADD COLUMN privacy VARCHAR(10) NOT NULL DEFAULT 'private'
    CHECK (privacy IN ('public', 'private'));

ALTER TABLE boards ADD CONSTRAINT chk_boards_privacy_shape CHECK (
  (privacy = 'public'  AND owner_user_id IS NULL     AND join_code IS NULL)
  OR
  (privacy = 'private' AND owner_user_id IS NOT NULL AND join_code IS NOT NULL)
);

INSERT INTO boards (name, privacy, owner_user_id, join_code)
VALUES ('Global', 'public', NULL, NULL);
