-- CC Signals: Replace boolean allow_training with enum for granular consent
-- Implements opt-in model: NULL = blocked, explicit signal = allowed

-- ============================================================================
-- CREATE CC SIGNAL ENUM TYPE
-- ============================================================================

CREATE TYPE cc_signal_type AS ENUM (
    'cc-cr',   -- Credit: Allow AI use with attribution
    'cc-dc',   -- Credit + Direct: Attribution + financial/in-kind support
    'cc-ec',   -- Credit + Ecosystem: Attribution + contribute to commons
    'cc-op',   -- Credit + Open: Attribution + keep derivatives open
    'no-ai'    -- No AI: Explicitly opt-out of AI training
);

COMMENT ON TYPE cc_signal_type IS 'Creative Commons Signals for AI training consent';

-- ============================================================================
-- ADD CC_SIGNAL COLUMN TO USER_STRUDELS
-- ============================================================================

ALTER TABLE user_strudels
ADD COLUMN IF NOT EXISTS cc_signal cc_signal_type;

COMMENT ON COLUMN user_strudels.cc_signal IS 'CC Signal for AI training consent. NULL = no preference (blocked in opt-in model)';

-- Index for filtering by signal
CREATE INDEX IF NOT EXISTS idx_user_strudels_cc_signal
ON user_strudels(cc_signal)
WHERE cc_signal IS NOT NULL AND cc_signal != 'no-ai';

-- ============================================================================
-- MIGRATE EXISTING DATA FROM allow_training
-- ============================================================================

UPDATE user_strudels
SET cc_signal = CASE
    WHEN allow_training = true THEN 'cc-cr'::cc_signal_type
    WHEN allow_training = false THEN 'no-ai'::cc_signal_type
    ELSE NULL
END
WHERE cc_signal IS NULL;

-- ============================================================================
-- UPDATE SEARCH FUNCTION FOR OPT-IN FILTERING
-- ============================================================================

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
    WHERE us.cc_signal IS NOT NULL          -- opt-in: must have signal
      AND us.cc_signal != 'no-ai'           -- not explicitly blocked
      AND us.use_in_training = true         -- admin curation
      AND us.is_public = true
      AND us.embedding IS NOT NULL
      AND u.training_consent = true         -- user global consent
    ORDER BY us.embedding <=> query_embedding
    LIMIT match_count;
END;
$$;

COMMENT ON FUNCTION search_user_strudels IS 'Search trainable user strudels by vector similarity (requires cc_signal + use_in_training + is_public + user.training_consent)';

-- ============================================================================
-- UPDATE TRIGGER TO HANDLE CC_SIGNAL
-- ============================================================================

CREATE OR REPLACE FUNCTION user_strudels_training_check_trigger() RETURNS trigger AS $$
BEGIN
  -- If strudel is being made private, clear cc_signal and use_in_training
  IF NEW.is_public = false THEN
    NEW.cc_signal := NULL;
    NEW.use_in_training := false;
  END IF;
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Trigger already exists, just updated the function above

-- ============================================================================
-- NOTE: allow_training column is kept for backward compatibility
-- It will be removed in a follow-up migration after code is updated
-- ============================================================================
