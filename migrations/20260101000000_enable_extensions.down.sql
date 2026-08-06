-- ==============================================================================
-- Migration: Rollback extensions
-- ==============================================================================

DROP FUNCTION IF EXISTS uuid_generate_v7();
DROP EXTENSION IF EXISTS citext;
