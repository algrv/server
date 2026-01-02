-- Add is_admin flag for role-based admin access

ALTER TABLE users
ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false;

COMMENT ON COLUMN users.is_admin IS 'Whether user has admin privileges';

-- Index for quick admin lookups
CREATE INDEX IF NOT EXISTS idx_users_is_admin
ON users(is_admin)
WHERE is_admin = true;
