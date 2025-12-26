-- Add users and user_strudels tables for authentication and saving user code

-- ============================================================================
-- USERS TABLE
-- ============================================================================

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  provider TEXT NOT NULL,           -- "google", "github", "apple"
  provider_id TEXT NOT NULL,        -- ID from the OAuth provider
  name TEXT,
  avatar_url TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(provider, provider_id)    -- Each provider+provider_id combo is unique
);

CREATE INDEX idx_users_provider ON users(provider, provider_id);
CREATE INDEX idx_users_email ON users(email);

-- ============================================================================
-- USER STRUDELS TABLE
-- ============================================================================

CREATE TABLE user_strudels (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  code TEXT NOT NULL,
  is_public BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_strudels_user_id ON user_strudels(user_id);
CREATE INDEX idx_user_strudels_created_at ON user_strudels(created_at DESC);
CREATE INDEX idx_user_strudels_public ON user_strudels(is_public) WHERE is_public = true;

-- ============================================================================
-- TRIGGER TO AUTO-UPDATE updated_at
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_strudels_updated_at
  BEFORE UPDATE ON user_strudels
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
