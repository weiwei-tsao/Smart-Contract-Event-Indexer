-- Initial database schema for Smart Contract Event Indexer

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: contracts
-- Stores monitored smart contracts
CREATE TABLE contracts (
    id SERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL UNIQUE,
    abi TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_block BIGINT NOT NULL DEFAULT 0,
    current_block BIGINT NOT NULL DEFAULT 0,
    confirm_blocks INTEGER NOT NULL DEFAULT 6 CHECK (confirm_blocks >= 1 AND confirm_blocks <= 100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX idx_contracts_address ON contracts(address);
CREATE INDEX idx_contracts_created_at ON contracts(created_at DESC);

-- Table: events
-- Stores indexed blockchain events
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    contract_address VARCHAR(42) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66) NOT NULL,
    transaction_hash VARCHAR(66) NOT NULL,
    transaction_index INTEGER NOT NULL,
    log_index INTEGER NOT NULL,
    args JSONB NOT NULL DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Ensure uniqueness: same event cannot be indexed twice
    UNIQUE(transaction_hash, log_index)
);

-- Indexes for common query patterns
CREATE INDEX idx_events_contract_address ON events(contract_address);
CREATE INDEX idx_events_block_number ON events(block_number DESC);
CREATE INDEX idx_events_contract_block ON events(contract_address, block_number DESC);
CREATE INDEX idx_events_transaction_hash ON events(transaction_hash);
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_events_event_name ON events(event_name);

-- GIN index for JSONB queries (MVP Phase 2)
CREATE INDEX idx_events_args_gin ON events USING GIN (args);

-- Table: indexer_state
-- Tracks indexing progress for each contract
CREATE TABLE indexer_state (
    contract_address VARCHAR(42) PRIMARY KEY,
    last_indexed_block BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (contract_address) REFERENCES contracts(address) ON DELETE CASCADE
);

-- Index for faster updates
CREATE INDEX idx_indexer_state_updated_at ON indexer_state(updated_at DESC);

-- Table: block_cache
-- Caches recent blocks for reorg detection
CREATE TABLE block_cache (
    block_number BIGINT PRIMARY KEY,
    block_hash VARCHAR(66) NOT NULL,
    parent_hash VARCHAR(66) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    cached_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for reorg detection queries
CREATE INDEX idx_block_cache_block_number ON block_cache(block_number DESC);
CREATE INDEX idx_block_cache_cached_at ON block_cache(cached_at DESC);

-- Table: backfill_jobs
-- Tracks historical data backfill jobs
CREATE TABLE backfill_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_address VARCHAR(42) NOT NULL,
    from_block BIGINT NOT NULL,
    to_block BIGINT NOT NULL,
    current_block BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    
    FOREIGN KEY (contract_address) REFERENCES contracts(address) ON DELETE CASCADE
);

-- Indexes for backfill job queries
CREATE INDEX idx_backfill_jobs_contract_address ON backfill_jobs(contract_address);
CREATE INDEX idx_backfill_jobs_status ON backfill_jobs(status);
CREATE INDEX idx_backfill_jobs_created_at ON backfill_jobs(created_at DESC);

-- Function: update_updated_at_column
-- Automatically updates the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic updated_at updates
CREATE TRIGGER update_contracts_updated_at BEFORE UPDATE ON contracts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_indexer_state_updated_at BEFORE UPDATE ON indexer_state
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_backfill_jobs_updated_at BEFORE UPDATE ON backfill_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create a view for contract statistics
CREATE OR REPLACE VIEW contract_stats AS
SELECT 
    c.address as contract_address,
    c.name as contract_name,
    COUNT(e.id) as total_events,
    MAX(e.block_number) as latest_event_block,
    c.current_block,
    c.current_block - COALESCE(MAX(e.block_number), c.start_block) as indexer_delay,
    MAX(e.timestamp) as last_event_time,
    c.updated_at
FROM contracts c
LEFT JOIN events e ON c.address = e.contract_address
GROUP BY c.address, c.name, c.current_block, c.start_block, c.updated_at;

-- Grant permissions (adjust as needed for your setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO indexer_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO indexer_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO indexer_user;

-- Add comments for documentation
COMMENT ON TABLE contracts IS 'Stores smart contracts being monitored by the indexer';
COMMENT ON TABLE events IS 'Stores indexed blockchain events with their arguments';
COMMENT ON TABLE indexer_state IS 'Tracks the indexing progress for each contract';
COMMENT ON TABLE block_cache IS 'Caches recent blocks for chain reorganization detection';
COMMENT ON TABLE backfill_jobs IS 'Tracks historical data backfill jobs';
COMMENT ON COLUMN contracts.confirm_blocks IS 'Number of confirmation blocks required (1-100)';
COMMENT ON COLUMN events.args IS 'Event arguments stored as JSONB for flexible querying';

