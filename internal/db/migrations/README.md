# Database Migrations

This directory contains database migration scripts for the TunnelBroker project.

## Migration 001: WireGuard Support

This migration adds WireGuard tunneling support to the existing database schema.

### What it does

- Adds 5 new columns to the `tunnels` table:
  - `server_private_key` (TEXT) - WireGuard server private key (base64)
  - `server_public_key` (TEXT) - WireGuard server public key (base64)
  - `client_private_key` (TEXT) - WireGuard client private key (base64)
  - `client_public_key` (TEXT) - WireGuard client public key (base64)
  - `listen_port` (INTEGER) - WireGuard listen port (51820-51821)
- Updates the type constraint to accept `'wg'` in addition to `'sit'` and `'gre'`
- **Does NOT modify or delete any existing data**

### Safety Features

1. Uses `ADD COLUMN IF NOT EXISTS` to prevent errors on re-run
2. New columns are nullable - existing tunnels are not affected
3. Creates backup before applying migration
4. Verifies data integrity after migration

## How to Apply Migration

### Option 1: Using the automated script (Recommended)

```bash
cd /home/user/tunnelbroker
./scripts/apply_wireguard_migration.sh
```

The script will:
- Test database connection
- Create automatic backup
- Apply migration
- Verify the changes
- Confirm data integrity

### Option 2: Manual application

```bash
# 1. Create backup first
pg_dump -h localhost -U your_user -d your_db \
  --table=public.tunnels --table=public.users \
  --data-only > backup.sql

# 2. Apply migration
psql -h localhost -U your_user -d your_db \
  -f internal/db/migrations/001_add_wireguard_support.sql

# 3. Verify migration
psql -h localhost -U your_user -d your_db \
  -f internal/db/migrations/verify_migration.sql
```

### Option 3: Using Supabase Dashboard

If you're using Supabase:

1. Go to SQL Editor in Supabase Dashboard
2. Copy contents of `001_add_wireguard_support.sql`
3. Paste and run the SQL
4. Verify by running `verify_migration.sql`

## Verification

After migration, run the verification script:

```bash
psql -h localhost -U your_user -d your_db \
  -f internal/db/migrations/verify_migration.sql
```

Expected results:
- 5 new columns should be present
- Type constraint should include 'sit', 'gre', and 'wg'
- All existing tunnels should be intact
- Tunnel count should match before/after

## Rollback

If you need to revert the migration:

⚠️ **WARNING**: This will remove all WireGuard tunnel data!

```bash
# 1. First, delete any WireGuard tunnels
psql -h localhost -U your_user -d your_db \
  -c "DELETE FROM public.tunnels WHERE type = 'wg';"

# 2. Then apply rollback
psql -h localhost -U your_user -d your_db \
  -f internal/db/migrations/001_add_wireguard_support_rollback.sql
```

## Testing Migration

You can test the migration on a development database:

```bash
# Create test database
createdb tunnelbroker_test

# Restore production backup to test
psql -d tunnelbroker_test < production_backup.sql

# Apply migration to test database
psql -d tunnelbroker_test \
  -f internal/db/migrations/001_add_wireguard_support.sql

# Verify
psql -d tunnelbroker_test \
  -f internal/db/migrations/verify_migration.sql

# If satisfied, apply to production
```

## Post-Migration Steps

After successful migration:

1. **Restart services**:
   ```bash
   sudo systemctl restart tunnelbroker
   sudo systemctl restart tunnelrecovery
   ```

2. **Test WireGuard tunnel creation**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/tunnels \
     -H "X-API-Key: your_key" \
     -H "Content-Type: application/json" \
     -d '{
       "type": "wg",
       "user_id": "test",
       "client_ipv4": "1.2.3.4"
     }'
   ```

3. **Monitor logs**:
   ```bash
   sudo journalctl -u tunnelbroker -f
   ```

## Troubleshooting

### Migration fails with "relation does not exist"

Ensure you're running the migration on the correct database and schema.

### Type constraint violation

If you have existing tunnels with invalid types, clean them up first:
```sql
SELECT * FROM public.tunnels WHERE type NOT IN ('sit', 'gre');
```

### Permission denied

Ensure the database user has ALTER TABLE privileges:
```sql
GRANT ALTER ON TABLE public.tunnels TO your_user;
```

### Backup restoration

If something goes wrong, restore from backup:
```bash
psql -d your_db < backup_before_wg_migration_YYYYMMDD_HHMMSS.sql
```

## Migration History

| Version | Date | Description | Applied |
|---------|------|-------------|---------|
| 001 | 2024-11-12 | Add WireGuard support | ⏳ Pending |

## Notes

- This migration is idempotent - safe to run multiple times
- No downtime required - existing tunnels continue to work
- New WireGuard tunnels can be created immediately after migration
- Existing SIT/GRE tunnels are completely unaffected
