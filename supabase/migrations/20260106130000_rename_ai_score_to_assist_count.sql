-- rename ai_contribution_score to ai_assist_count (simpler metric)
-- drop old index first
DROP INDEX IF EXISTS idx_user_strudels_ai_score;

-- drop old column and add new one with correct type
ALTER TABLE user_strudels
DROP COLUMN IF EXISTS ai_contribution_score;

ALTER TABLE user_strudels
ADD COLUMN IF NOT EXISTS ai_assist_count INTEGER DEFAULT 0;

COMMENT ON COLUMN user_strudels.ai_assist_count IS 'Count of AI code responses used to create this strudel';

CREATE INDEX IF NOT EXISTS idx_user_strudels_ai_assist
ON user_strudels(ai_assist_count)
WHERE ai_assist_count > 0;
