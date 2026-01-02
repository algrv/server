-- Rename allow_training to use_in_training
-- This field is admin-controlled for curated training data

ALTER TABLE user_strudels
RENAME COLUMN allow_training TO use_in_training;

-- Update the index name
DROP INDEX IF EXISTS idx_user_strudels_training;
CREATE INDEX idx_user_strudels_use_in_training
ON user_strudels(use_in_training)
WHERE use_in_training = true;

-- Update comments
COMMENT ON COLUMN user_strudels.use_in_training IS 'Admin-controlled flag for curated training data (requires user consent + is_public)';

-- Update the search function to use new column name
CREATE OR REPLACE FUNCTION search_user_strudels(
    query_embedding extensions.vector(1536),
    match_count int DEFAULT 3
)
RETURNS TABLE (
    id UUID,
    title TEXT,
    description TEXT,
    code TEXT,
    tags TEXT[],
    user_id UUID,
    similarity FLOAT
)
LANGUAGE plpgsql STABLE
AS $$
BEGIN
    PERFORM set_config('search_path', 'extensions, public', true);

    RETURN QUERY
    SELECT
        us.id,
        us.title,
        us.description,
        us.code,
        us.tags,
        us.user_id,
        1 - (us.embedding <=> query_embedding) AS similarity
    FROM user_strudels us
    INNER JOIN users u ON us.user_id = u.id
    WHERE us.use_in_training = true
      AND us.is_public = true
      AND us.embedding IS NOT NULL
      AND u.training_consent = true
    ORDER BY us.embedding <=> query_embedding
    LIMIT match_count;
END;
$$;

-- Update the trigger function to use new column name
CREATE OR REPLACE FUNCTION user_strudels_training_check_trigger() RETURNS trigger AS $$
BEGIN
  -- If strudel is being made private, disable training
  IF NEW.is_public = false AND NEW.use_in_training = true THEN
    NEW.use_in_training := false;
  END IF;
  RETURN NEW;
END
$$ LANGUAGE plpgsql;
