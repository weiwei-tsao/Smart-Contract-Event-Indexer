-- Remove columns added in migration 002

DROP INDEX IF EXISTS idx_indexer_state_status;

ALTER TABLE indexer_state DROP COLUMN IF EXISTS id;
ALTER TABLE indexer_state DROP COLUMN IF EXISTS last_block_hash;
ALTER TABLE indexer_state DROP COLUMN IF EXISTS last_processed_at;
ALTER TABLE indexer_state DROP COLUMN IF EXISTS status;
ALTER TABLE indexer_state DROP COLUMN IF EXISTS error_count;
ALTER TABLE indexer_state DROP COLUMN IF EXISTS last_error;
ALTER TABLE indexer_state DROP COLUMN IF EXISTS created_at;
