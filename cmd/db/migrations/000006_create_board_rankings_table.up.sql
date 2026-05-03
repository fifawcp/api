CREATE TABLE IF NOT EXISTS board_rankings (
  board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  total_points INT NOT NULL DEFAULT 0,
  global_points INT NOT NULL DEFAULT 0,
  detailed_points INT NOT NULL DEFAULT 0,
  exact_hits INT NOT NULL DEFAULT 0,
  correct_outcomes INT NOT NULL DEFAULT 0,
  updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  PRIMARY KEY (board_id, user_id),
  CONSTRAINT fk_board_rankings_board_member
    FOREIGN KEY (board_id, user_id)
    REFERENCES board_members(board_id, user_id)
    ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_board_rankings_rank_order
ON board_rankings(board_id, total_points DESC, updated_at ASC, user_id ASC);