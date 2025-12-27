-- Migration: Separate Anonymous Participants
-- Description: Split authenticated and anonymous participants into separate tables for better security and data integrity
-- Created: 2025-12-27

-- Create anonymous_participants table
CREATE TABLE anonymous_participants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  display_name TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('co-author', 'viewer')),  -- Anonymous users cannot be hosts
  status TEXT NOT NULL CHECK (status IN ('active', 'left')) DEFAULT 'active',
  joined_at TIMESTAMPTZ DEFAULT NOW(),
  left_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '24 hours'  -- Auto-expire after 24 hours
);

CREATE INDEX idx_anonymous_participants_session ON anonymous_participants(session_id);
CREATE INDEX idx_anonymous_participants_expires ON anonymous_participants(expires_at) WHERE status = 'active';

-- Migrate existing NULL user_id rows to anonymous_participants
INSERT INTO anonymous_participants (session_id, display_name, role, status, joined_at, left_at)
SELECT
  session_id,
  COALESCE(display_name, 'Anonymous User'),  -- Provide default if NULL
  role,
  status,
  joined_at,
  left_at
FROM session_participants
WHERE user_id IS NULL;

-- Delete migrated rows from session_participants
DELETE FROM session_participants WHERE user_id IS NULL;

-- Add NOT NULL constraints now that NULLs are removed
ALTER TABLE session_participants ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE session_participants ALTER COLUMN display_name SET NOT NULL;

-- Update comments
COMMENT ON TABLE anonymous_participants IS 'Temporary anonymous participants who join via invite tokens without creating accounts';
COMMENT ON TABLE session_participants IS 'Authenticated users participating in sessions with full user accounts';

COMMENT ON COLUMN anonymous_participants.role IS 'co-author: can edit code, viewer: read-only. Note: anonymous users cannot be hosts';
COMMENT ON COLUMN anonymous_participants.expires_at IS 'Anonymous participants automatically expire after 24 hours for cleanup';
