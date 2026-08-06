-- ==============================================================================
-- Migration: Enable required PostgreSQL extensions
-- ==============================================================================
-- UUID v7 generation function (pg_uuidv7 extension or custom PL/pgSQL)
-- CITEXT extension for case-insensitive text columns

-- +migrate Up

CREATE EXTENSION IF NOT EXISTS citext;

-- uuid_generate_v7() — Custom PL/pgSQL implementation
-- Generates UUIDv7 (RFC 9562) with millisecond-precision timestamp prefix
-- for time-ordered, index-friendly primary keys (Playbook §7.6).
CREATE OR REPLACE FUNCTION uuid_generate_v7() RETURNS uuid AS $$
DECLARE
    unix_ts_ms BIGINT;
    buffer     BYTEA;
BEGIN
    unix_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;
    buffer := E'\\x'
        || lpad(to_hex(unix_ts_ms >> 16), 8, '0')
        || lpad(to_hex(unix_ts_ms & x'FFFF'::INT), 4, '0')
        || lpad(to_hex((random() * x'0FFF'::INT)::INT | x'7000'::INT), 4, '0')
        || lpad(to_hex((random() * x'3FFFFFFFFFFFFFFF'::BIGINT)::BIGINT | (x'8000000000000000'::BIGINT)), 16, '0');
    RETURN encode(buffer, 'hex')::uuid;
END;
$$ LANGUAGE plpgsql VOLATILE;

