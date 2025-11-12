#!/bin/bash
set -e

# Script to apply WireGuard migration to PostgreSQL database
# This script safely adds WireGuard support to the tunnels table

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
MIGRATION_FILE="internal/db/migrations/001_add_wireguard_support.sql"
VERIFY_FILE="internal/db/migrations/verify_migration.sql"
BACKUP_FILE="backup_before_wg_migration_$(date +%Y%m%d_%H%M%S).sql"

echo -e "${GREEN}=== WireGuard Migration Script ===${NC}"
echo ""

# Check if migration file exists
if [ ! -f "$MIGRATION_FILE" ]; then
    echo -e "${RED}ERROR: Migration file not found: $MIGRATION_FILE${NC}"
    exit 1
fi

# Get database connection details
echo "Please provide database connection details:"
read -p "Database host [localhost]: " DB_HOST
DB_HOST=${DB_HOST:-localhost}

read -p "Database port [5432]: " DB_PORT
DB_PORT=${DB_PORT:-5432}

read -p "Database name: " DB_NAME
if [ -z "$DB_NAME" ]; then
    echo -e "${RED}ERROR: Database name is required${NC}"
    exit 1
fi

read -p "Database user: " DB_USER
if [ -z "$DB_USER" ]; then
    echo -e "${RED}ERROR: Database user is required${NC}"
    exit 1
fi

read -sp "Database password: " DB_PASSWORD
echo ""

export PGPASSWORD="$DB_PASSWORD"

# Test connection
echo -e "\n${YELLOW}Testing database connection...${NC}"
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Cannot connect to database${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Database connection successful${NC}"

# Backup database
echo -e "\n${YELLOW}Creating database backup...${NC}"
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    --table=public.tunnels --table=public.users \
    --data-only --file="$BACKUP_FILE"

if [ -f "$BACKUP_FILE" ]; then
    echo -e "${GREEN}✓ Backup created: $BACKUP_FILE${NC}"
else
    echo -e "${RED}ERROR: Backup creation failed${NC}"
    exit 1
fi

# Count existing tunnels
TUNNEL_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -t -c "SELECT COUNT(*) FROM public.tunnels;" | xargs)
echo -e "\n${YELLOW}Existing tunnels in database: $TUNNEL_COUNT${NC}"

# Ask for confirmation
echo -e "\n${YELLOW}This will:${NC}"
echo "  1. Add WireGuard columns to the tunnels table"
echo "  2. Update type constraint to accept 'wg' type"
echo "  3. Keep all existing data intact"
echo ""
read -p "Do you want to proceed? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo -e "${YELLOW}Migration cancelled${NC}"
    exit 0
fi

# Apply migration
echo -e "\n${YELLOW}Applying migration...${NC}"
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$MIGRATION_FILE"; then
    echo -e "${GREEN}✓ Migration applied successfully${NC}"
else
    echo -e "${RED}ERROR: Migration failed${NC}"
    echo -e "${YELLOW}You can restore from backup: $BACKUP_FILE${NC}"
    exit 1
fi

# Verify migration
if [ -f "$VERIFY_FILE" ]; then
    echo -e "\n${YELLOW}Verifying migration...${NC}"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$VERIFY_FILE"
fi

# Count tunnels after migration
TUNNEL_COUNT_AFTER=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -t -c "SELECT COUNT(*) FROM public.tunnels;" | xargs)

if [ "$TUNNEL_COUNT" -eq "$TUNNEL_COUNT_AFTER" ]; then
    echo -e "\n${GREEN}✓ Data integrity verified - all $TUNNEL_COUNT tunnels preserved${NC}"
else
    echo -e "\n${RED}WARNING: Tunnel count mismatch!${NC}"
    echo -e "  Before: $TUNNEL_COUNT"
    echo -e "  After: $TUNNEL_COUNT_AFTER"
fi

# Cleanup
unset PGPASSWORD

echo -e "\n${GREEN}=== Migration Complete ===${NC}"
echo -e "Backup file: ${YELLOW}$BACKUP_FILE${NC}"
echo -e "\nYou can now create WireGuard tunnels using type='wg'"
echo -e "\nTo rollback this migration, run:"
echo -e "  ${YELLOW}psql ... -f internal/db/migrations/001_add_wireguard_support_rollback.sql${NC}"
