-- Add forked_from column to track fork relationships
ALTER TABLE user_strudels
ADD COLUMN IF NOT EXISTS forked_from UUID REFERENCES user_strudels(id) ON DELETE SET NULL;

-- Index for finding forks of a strudel
CREATE INDEX IF NOT EXISTS idx_user_strudels_forked_from
ON user_strudels(forked_from) WHERE forked_from IS NOT NULL;

COMMENT ON COLUMN user_strudels.forked_from IS 'ID of the original strudel this was forked from';
