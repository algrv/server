-- Add conversation_history to user_strudels for contextual AI interactions

ALTER TABLE user_strudels
  ADD COLUMN conversation_history JSONB DEFAULT '[]'::jsonb;

COMMENT ON COLUMN user_strudels.conversation_history IS 'Stores the conversation history as an array of {role, content} objects for contextual code generation';
