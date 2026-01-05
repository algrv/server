-- add clarifying_questions column to strudel_messages for storing non-code responses
ALTER TABLE strudel_messages
ADD COLUMN clarifying_questions JSONB DEFAULT NULL;

COMMENT ON COLUMN strudel_messages.clarifying_questions IS 'array of clarifying questions for non-code responses';
