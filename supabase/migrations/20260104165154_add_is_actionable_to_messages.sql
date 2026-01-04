-- Add is_actionable column to session_messages for tracking actionable AI responses

ALTER TABLE session_messages
  ADD COLUMN is_actionable BOOLEAN DEFAULT false;

COMMENT ON COLUMN session_messages.is_actionable IS 'Whether this AI response contains actionable code that can be applied';
