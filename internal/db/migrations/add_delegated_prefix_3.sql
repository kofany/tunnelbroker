-- Migration: Add third delegated prefix to tunnels table
-- Date: 2024-07-01
-- Description: This migration adds support for a third IPv6 prefix delegation
-- from a dedicated /48 range to each tunnel.

-- Step 1: Add the new column to store the third delegated prefix
ALTER TABLE public.tunnels ADD COLUMN delegated_prefix_3 text;

-- Step 2: Add an index to improve query performance when checking for prefix uniqueness
CREATE INDEX idx_tunnels_delegated_prefix_3 ON public.tunnels (delegated_prefix_3);

-- Step 3: Add a comment to document the column's purpose
COMMENT ON COLUMN public.tunnels.delegated_prefix_3 IS 'Third delegated /64 prefix from dedicated /48 range';

-- Note: Existing tunnels will have NULL for delegated_prefix_3
-- They will continue to work with just two prefixes for backward compatibility
