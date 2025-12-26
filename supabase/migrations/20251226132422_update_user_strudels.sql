-- Add description, tags, and categories to user_strudels table

ALTER TABLE user_strudels
  ADD COLUMN description TEXT,
  ADD COLUMN tags TEXT[],
  ADD COLUMN categories TEXT[];

-- Create GIN index for tags array for efficient searching
CREATE INDEX idx_user_strudels_tags ON user_strudels USING GIN (tags);

-- Create GIN index for categories array for efficient searching
CREATE INDEX idx_user_strudels_categories ON user_strudels USING GIN (categories);
