-- ==============================================================================
-- Migration: Drop sessions table
-- ==============================================================================

DROP INDEX IF EXISTS idx_sessions_user_id_status;
DROP TABLE IF EXISTS sessions;
