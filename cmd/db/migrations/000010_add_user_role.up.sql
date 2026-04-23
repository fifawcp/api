ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'user';
ALTER TABLE users ADD CONSTRAINT check_user_role CHECK (role IN ('user', 'admin'));
