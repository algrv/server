-- Add license field for Creative Commons licenses
-- License determines general sharing rights, cc_signal handles AI-specific preferences

-- ============================================================================
-- ADD LICENSE COLUMN TO USER_STRUDELS
-- ============================================================================

ALTER TABLE user_strudels
ADD COLUMN IF NOT EXISTS license TEXT;

COMMENT ON COLUMN user_strudels.license IS 'Creative Commons license (e.g., CC BY-NC-SA 4.0). Determines sharing rights.';

-- ============================================================================
-- VALID LICENSE VALUES (for reference, not enforced at DB level)
-- ============================================================================
-- CC0 1.0        - Public Domain
-- CC BY 4.0      - Attribution
-- CC BY-SA 4.0   - Attribution-ShareAlike
-- CC BY-NC 4.0   - Attribution-NonCommercial
-- CC BY-NC-SA 4.0 - Attribution-NonCommercial-ShareAlike
-- CC BY-ND 4.0   - Attribution-NoDerivatives
-- CC BY-NC-ND 4.0 - Attribution-NonCommercial-NoDerivatives
-- ============================================================================
