-- Migration: Add System User for Anonymous Sessions
-- Description: Creates a system user that acts as host for anonymous sessions
-- Created: 2025-12-29

-- Insert system user for anonymous sessions
-- Using the nil UUID (well-known pattern for system/default entities)
-- This user cannot be authenticated via OAuth (provider: "system" is not valid)
INSERT INTO users (id, email, provider, provider_id, name, tier)
VALUES (
  '00000000-0000-0000-0000-000000000000',
  'system@algojams.local',
  'system',
  'system',
  'System',
  'free'
) ON CONFLICT (id) DO NOTHING;

-- Prevent accidental deletion of system user
CREATE OR REPLACE FUNCTION prevent_system_user_deletion()
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.id = '00000000-0000-0000-0000-000000000000' THEN
    RAISE EXCEPTION 'Cannot delete system user - required for anonymous sessions';
  END IF;
  RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS protect_system_user ON users;
CREATE TRIGGER protect_system_user
  BEFORE DELETE ON users
  FOR EACH ROW
  EXECUTE FUNCTION prevent_system_user_deletion();

COMMENT ON TABLE users IS 'User accounts. ID 00000000-0000-0000-0000-000000000000 is the system user for anonymous sessions.';
