-- Migration to add WireGuard support to tunnels table
-- This adds fields for WireGuard keys and port, and updates the type constraint

-- Add WireGuard specific columns to tunnels table
ALTER TABLE public.tunnels
ADD COLUMN IF NOT EXISTS server_private_key TEXT,
ADD COLUMN IF NOT EXISTS server_public_key TEXT,
ADD COLUMN IF NOT EXISTS client_private_key TEXT,
ADD COLUMN IF NOT EXISTS client_public_key TEXT,
ADD COLUMN IF NOT EXISTS listen_port INTEGER;

-- Drop old type constraint
ALTER TABLE public.tunnels DROP CONSTRAINT IF EXISTS tunnels_type_check;

-- Add new type constraint that includes 'wg'
ALTER TABLE public.tunnels
ADD CONSTRAINT tunnels_type_check
CHECK (type = ANY (ARRAY['sit'::text, 'gre'::text, 'wg'::text]));

-- Add comments for new columns
COMMENT ON COLUMN public.tunnels.server_private_key IS 'WireGuard server private key (base64 encoded)';
COMMENT ON COLUMN public.tunnels.server_public_key IS 'WireGuard server public key (base64 encoded)';
COMMENT ON COLUMN public.tunnels.client_private_key IS 'WireGuard client private key (base64 encoded)';
COMMENT ON COLUMN public.tunnels.client_public_key IS 'WireGuard client public key (base64 encoded)';
COMMENT ON COLUMN public.tunnels.listen_port IS 'WireGuard listen port (typically 51820-51821)';
