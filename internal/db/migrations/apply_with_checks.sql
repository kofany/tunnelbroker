-- Complete migration script with pre/post checks
-- Safe to run on production - includes verification and rollback info

\echo '================================================'
\echo 'WireGuard Migration - Starting'
\echo '================================================'
\echo ''

-- PRE-MIGRATION CHECKS
\echo 'PRE-MIGRATION CHECKS:'
\echo '--------------------'

-- Check current tunnel count
\echo 'Current tunnel count:'
SELECT
    COUNT(*) AS total_tunnels,
    COUNT(*) FILTER (WHERE type = 'sit') AS sit_tunnels,
    COUNT(*) FILTER (WHERE type = 'gre') AS gre_tunnels
FROM public.tunnels;

\echo ''
\echo 'Current table structure:'
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'tunnels'
ORDER BY ordinal_position;

\echo ''
\echo 'Press Ctrl+C to abort, or press Enter to continue...'
\prompt 'Continue? ' dummy

-- BEGIN TRANSACTION (optional - uncomment for transactional migration)
-- BEGIN;

\echo ''
\echo 'APPLYING MIGRATION:'
\echo '-------------------'

-- Add WireGuard specific columns to tunnels table
\echo 'Adding WireGuard columns...'
ALTER TABLE public.tunnels
ADD COLUMN IF NOT EXISTS server_private_key TEXT,
ADD COLUMN IF NOT EXISTS server_public_key TEXT,
ADD COLUMN IF NOT EXISTS client_private_key TEXT,
ADD COLUMN IF NOT EXISTS client_public_key TEXT,
ADD COLUMN IF NOT EXISTS listen_port INTEGER;

\echo 'Updating type constraint...'
-- Drop old type constraint
ALTER TABLE public.tunnels DROP CONSTRAINT IF EXISTS tunnels_type_check;

-- Add new type constraint that includes 'wg'
ALTER TABLE public.tunnels
ADD CONSTRAINT tunnels_type_check
CHECK (type = ANY (ARRAY['sit'::text, 'gre'::text, 'wg'::text]));

\echo 'Adding column comments...'
-- Add comments for new columns
COMMENT ON COLUMN public.tunnels.server_private_key IS 'WireGuard server private key (base64 encoded)';
COMMENT ON COLUMN public.tunnels.server_public_key IS 'WireGuard server public key (base64 encoded)';
COMMENT ON COLUMN public.tunnels.client_private_key IS 'WireGuard client private key (base64 encoded)';
COMMENT ON COLUMN public.tunnels.client_public_key IS 'WireGuard client public key (base64 encoded)';
COMMENT ON COLUMN public.tunnels.listen_port IS 'WireGuard listen port (typically 51820-51821)';

-- COMMIT; -- Uncomment if using transaction

\echo ''
\echo 'POST-MIGRATION VERIFICATION:'
\echo '----------------------------'

-- Verify columns were added
\echo 'Verifying new columns exist:'
SELECT
    column_name,
    data_type,
    is_nullable,
    CASE
        WHEN column_name IN ('server_private_key', 'server_public_key',
                            'client_private_key', 'client_public_key', 'listen_port')
        THEN '✓ NEW'
        ELSE ''
    END AS status
FROM information_schema.columns
WHERE table_schema = 'public'
    AND table_name = 'tunnels'
    AND column_name IN (
        'server_private_key', 'server_public_key',
        'client_private_key', 'client_public_key', 'listen_port'
    )
ORDER BY column_name;

\echo ''
\echo 'Verifying type constraint:'
SELECT
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid = 'public.tunnels'::regclass
    AND conname = 'tunnels_type_check';

\echo ''
\echo 'Verifying data integrity:'
SELECT
    COUNT(*) AS total_tunnels,
    COUNT(*) FILTER (WHERE type = 'sit') AS sit_tunnels,
    COUNT(*) FILTER (WHERE type = 'gre') AS gre_tunnels,
    COUNT(*) FILTER (WHERE type = 'wg') AS wg_tunnels,
    COUNT(*) FILTER (WHERE server_private_key IS NOT NULL) AS with_wg_keys
FROM public.tunnels;

\echo ''
\echo '================================================'
\echo 'Migration Complete!'
\echo '================================================'
\echo ''
\echo 'Next steps:'
\echo '1. Restart tunnelbroker service: sudo systemctl restart tunnelbroker'
\echo '2. Restart tunnelrecovery service: sudo systemctl restart tunnelrecovery'
\echo '3. Test creating a WireGuard tunnel with type="wg"'
\echo ''
\echo 'To rollback, run: internal/db/migrations/001_add_wireguard_support_rollback.sql'
\echo ''
