CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_info JSONB,
    ip_address TEXT,
    user_agent TEXT,
    last_used_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP(0) WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT check_expires_after_created CHECK (expires_at > created_at),
    CONSTRAINT check_last_used_before_expires CHECK (last_used_at <= expires_at)
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);