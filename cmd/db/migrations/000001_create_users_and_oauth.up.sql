CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
  id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  first_name VARCHAR(255) NOT NULL,
  last_name  VARCHAR(255) NOT NULL,
  username   VARCHAR(20)  NOT NULL UNIQUE,
  email      CITEXT       NOT NULL UNIQUE,
  role       VARCHAR(20)  NOT NULL DEFAULT 'user',
  created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  CONSTRAINT check_users_email_lowercase CHECK (email = LOWER(TRIM(email))),
  CONSTRAINT check_user_role CHECK (role IN ('user', 'admin'))
);

CREATE INDEX IF NOT EXISTS idx_users_email    ON users(LOWER(TRIM(email)));
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

CREATE TABLE IF NOT EXISTS oauth_accounts (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  provider     TEXT NOT NULL,
  provider_sub TEXT NOT NULL,
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at   TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  UNIQUE (provider, provider_sub),
  UNIQUE (user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_oauth_accounts_user_id ON oauth_accounts(user_id);
