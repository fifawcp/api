CREATE TABLE IF NOT EXISTS boards (
  id         BIGSERIAL PRIMARY KEY,
  name       VARCHAR(20) NOT NULL,
  privacy    VARCHAR(10) NOT NULL DEFAULT 'private',
  join_code  VARCHAR(8),
  created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_boards_privacy CHECK (privacy IN ('public', 'private', 'global')),
  CONSTRAINT chk_boards_privacy_shape CHECK (
    (privacy = 'global'  AND join_code IS NULL)
    OR (privacy = 'public'  AND join_code IS NULL)
    OR (privacy = 'private' AND join_code IS NOT NULL)
  )
);

CREATE UNIQUE INDEX boards_join_code_unique ON boards(join_code) WHERE join_code IS NOT NULL;
CREATE UNIQUE INDEX idx_boards_one_global   ON boards(privacy)   WHERE privacy = 'global';
CREATE INDEX        idx_boards_created_at   ON boards(created_at DESC);

CREATE TABLE IF NOT EXISTS board_members (
  board_id   BIGINT      NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role       VARCHAR(20) NOT NULL DEFAULT 'member',
  created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (board_id, user_id),
  CONSTRAINT chk_board_members_role CHECK (role IN ('admin', 'member', 'owner'))
);

CREATE INDEX        idx_board_members_role      ON board_members(role);
CREATE INDEX        idx_board_members_user_id   ON board_members(user_id);
CREATE UNIQUE INDEX idx_board_members_one_owner ON board_members(board_id) WHERE role = 'owner';
