CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  first_name VARCHAR(255) NOT NULL,
  last_name VARCHAR(255) NOT NULL,
  username VARCHAR(50) NOT NULL UNIQUE,
  email CITEXT NOT NULL UNIQUE,
  created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
  CONSTRAINT check_users_email_lowercase CHECK (email = LOWER(TRIM(email)))
);

CREATE INDEX idx_users_email ON users(LOWER(TRIM(email)));
CREATE INDEX idx_users_username ON users(username);