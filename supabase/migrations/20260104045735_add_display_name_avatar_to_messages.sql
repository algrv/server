-- Add display_name and avatar_url to session_messages for chat history

ALTER TABLE session_messages
  ADD COLUMN display_name TEXT,
  ADD COLUMN avatar_url TEXT;

COMMENT ON COLUMN session_messages.display_name IS 'Display name of the message sender at the time of sending';
COMMENT ON COLUMN session_messages.avatar_url IS 'Avatar URL of the message sender at the time of sending';
