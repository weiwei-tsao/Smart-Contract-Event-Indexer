-- Add missing columns to indexer_state table

-- Add id column as auto-increment
ALTER TABLE indexer_state ADD COLUMN IF NOT EXISTS id BIGSERIAL;

-- Add last_block_hash column
ALTER TABLE indexer_state ADD COLUMN IF NOT EXISTS last_block_hash VARCHAR(66);

-- Add last_processed_at column
ALTER TABLE indexer_state ADD COLUMN IF NOT EXISTS last_processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Add status column
ALTER TABLE indexer_state ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';

-- Add error_count column
ALTER TABLE indexer_state ADD COLUMN IF NOT EXISTS error_count INTEGER DEFAULT 0;

-- Add last_error column
ALTER TABLE indexer_state ADD COLUMN IF NOT EXISTS last_error TEXT;

-- Add created_at column
ALTER TABLE indexer_state ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Add index on status for filtering
CREATE INDEX IF NOT EXISTS idx_indexer_state_status ON indexer_state(status);
