-- Create strudel_messages table for AI conversation history (strudel-scoped)
-- session_messages remains for chat only (session-scoped)

CREATE TABLE strudel_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  strudel_id UUID NOT NULL REFERENCES user_strudels(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
  content TEXT NOT NULL,
  is_actionable BOOLEAN DEFAULT false,
  is_code_response BOOLEAN DEFAULT true,
  display_name TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_strudel_messages_strudel ON strudel_messages(strudel_id, created_at);

COMMENT ON TABLE strudel_messages IS 'AI conversation history for saved strudels. Drafts use localStorage.';
COMMENT ON COLUMN strudel_messages.role IS 'user: user prompt, assistant: AI response';
COMMENT ON COLUMN strudel_messages.is_code_response IS 'Whether this response should update the editor';
