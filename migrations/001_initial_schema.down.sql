-- Rollback migration: Drop all tables and objects created in 001_initial_schema.up.sql

-- Drop views
DROP VIEW IF EXISTS contract_stats;

-- Drop triggers
DROP TRIGGER IF EXISTS update_contracts_updated_at ON contracts;
DROP TRIGGER IF EXISTS update_indexer_state_updated_at ON indexer_state;
DROP TRIGGER IF EXISTS update_backfill_jobs_updated_at ON backfill_jobs;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS backfill_jobs;
DROP TABLE IF EXISTS block_cache;
DROP TABLE IF EXISTS indexer_state;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS contracts;

-- Drop extensions (optional - only if not used by other databases)
-- DROP EXTENSION IF EXISTS "uuid-ossp";

