-- Default cc_signal to 'no-ai' (restrictive by default)
-- This ensures new strudels don't allow AI use unless explicitly set

-- Set default for new rows
ALTER TABLE user_strudels
ALTER COLUMN cc_signal SET DEFAULT 'no-ai'::cc_signal_type;

-- Update existing NULL values to 'no-ai'
UPDATE user_strudels
SET cc_signal = 'no-ai'::cc_signal_type
WHERE cc_signal IS NULL;
