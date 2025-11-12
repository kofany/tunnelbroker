-- Rollback migration for WireGuard support
-- WARNING: This will remove WireGuard columns and data!
-- Use only if you need to revert the migration

-- Remove WireGuard specific columns
ALTER TABLE public.tunnels
DROP COLUMN IF EXISTS server_private_key,
DROP COLUMN IF EXISTS server_public_key,
DROP COLUMN IF EXISTS client_private_key,
DROP COLUMN IF EXISTS client_public_key,
DROP COLUMN IF EXISTS listen_port;

-- Drop new type constraint
ALTER TABLE public.tunnels DROP CONSTRAINT IF EXISTS tunnels_type_check;

-- Restore old type constraint (only sit and gre)
ALTER TABLE public.tunnels
ADD CONSTRAINT tunnels_type_check
CHECK (type = ANY (ARRAY['sit'::text, 'gre'::text]));

-- Note: Any WireGuard tunnels (type='wg') will fail the constraint check
-- You must delete WireGuard tunnels before running this rollback:
-- DELETE FROM public.tunnels WHERE type = 'wg';
