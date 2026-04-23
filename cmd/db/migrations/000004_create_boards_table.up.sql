CREATE TABLE IF NOT EXISTS boards (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR(120) NOT NULL,
  owner_user_id UUID NOT NULL REFERENCES users(id),
  join_code VARCHAR(8) NOT NULL UNIQUE,
  created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_boards_owner_user_id ON boards(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_boards_created_at ON boards(created_at DESC);