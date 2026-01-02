-- Add is_discoverable column to sessions for "go live" feature
ALTER TABLE sessions ADD COLUMN is_discoverable BOOLEAN DEFAULT false;

-- Index for efficient querying of live discoverable sessions
CREATE INDEX idx_sessions_discoverable_active
ON sessions (is_discoverable, is_active)
WHERE is_discoverable = true AND is_active = true;
