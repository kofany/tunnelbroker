-- Verification script for WireGuard migration
-- Run this after applying the migration to verify it was successful

-- Check if new columns exist
SELECT
    column_name,
    data_type,
    is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
    AND table_name = 'tunnels'
    AND column_name IN (
        'server_private_key',
        'server_public_key',
        'client_private_key',
        'client_public_key',
        'listen_port'
    )
ORDER BY column_name;

-- Expected result: 5 rows showing the new columns

-- Check the type constraint
SELECT
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid = 'public.tunnels'::regclass
    AND conname = 'tunnels_type_check';

-- Expected: CHECK constraint should include 'sit', 'gre', and 'wg'

-- Check existing data is intact
SELECT
    COUNT(*) AS total_tunnels,
    COUNT(*) FILTER (WHERE type = 'sit') AS sit_tunnels,
    COUNT(*) FILTER (WHERE type = 'gre') AS gre_tunnels,
    COUNT(*) FILTER (WHERE type = 'wg') AS wg_tunnels,
    COUNT(*) FILTER (WHERE server_private_key IS NOT NULL) AS tunnels_with_wg_keys
FROM public.tunnels;

-- Expected: All existing SIT/GRE tunnels should be present
-- No data should be lost

-- Check column comments
SELECT
    column_name,
    col_description('public.tunnels'::regclass, ordinal_position) AS column_comment
FROM information_schema.columns
WHERE table_schema = 'public'
    AND table_name = 'tunnels'
    AND column_name IN (
        'server_private_key',
        'server_public_key',
        'client_private_key',
        'client_public_key',
        'listen_port'
    )
ORDER BY column_name;

-- Expected: Each column should have a descriptive comment
