CREATE TABLE IF NOT EXISTS board_members (
  board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role VARCHAR(20) NOT NULL DEFAULT 'member',
  created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (board_id, user_id),
  CONSTRAINT chk_board_members_role CHECK (role IN ('admin', 'member'))
);

CREATE INDEX IF NOT EXISTS idx_board_members_role    ON board_members(role);
CREATE INDEX IF NOT EXISTS idx_board_members_user_id ON board_members(user_id);