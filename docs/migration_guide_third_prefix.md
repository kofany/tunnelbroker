# Migration Guide: Adding Third IPv6 Prefix Support

This guide provides instructions for system administrators to migrate the TunnelBroker system to support a third IPv6 prefix delegation for each tunnel.

## Overview

The migration adds a third /64 prefix delegation from a dedicated /48 range to each tunnel, in addition to the existing two prefixes from paired /44 ranges.

## Prerequisites

- PostgreSQL database access with admin privileges
- Access to the TunnelBroker configuration files
- System downtime window (recommended)

## Migration Steps

### 1. Backup the Database

Before making any changes, create a backup of your database:

```bash
pg_dump -U your_db_user -h your_db_host -d your_db_name > tunnelbroker_backup_$(date +%Y%m%d).sql
```

### 2. Update Configuration

Add the new third prefix configuration to your `config.yaml` file:

```yaml
prefixes:
  para1:
    primary: "2a05:xxxx:xxxx::/44"
    secondary: "2a12:xxxx:xxxx::/44"
  para2:
    primary: "2a05:xxxx:xxxx::/44"
    secondary: "2a05:xxxx:xxxx::/44"
  ula: "fde4:xxxx:xxxx::/48"
  third: "2a06:xxxx:xxxx::/48"  # Add this line with your dedicated /48 prefix
```

### 3. Apply Database Migration

Run the database migration script:

```bash
psql -U your_db_user -h your_db_host -d your_db_name -f /path/to/internal/db/migrations/add_delegated_prefix_3.sql
```

The migration script will:
- Add the `delegated_prefix_3` column to the `tunnels` table
- Create an index for performance optimization
- Add documentation comments

### 4. Update TunnelBroker Software

Deploy the updated TunnelBroker software that includes support for the third prefix:

```bash
# Stop the service
systemctl stop tunnelbroker

# Deploy new version
cp /path/to/new/tunnelbroker /usr/local/bin/tunnelbroker
chmod +x /usr/local/bin/tunnelbroker

# Start the service
systemctl start tunnelbroker
```

### 5. Verify the Migration

Check that the system is working correctly:

1. Create a new tunnel and verify it receives three prefixes
2. Check the database to confirm the third prefix is stored correctly
3. Verify that existing tunnels continue to work with two prefixes

```bash
# Example database check
psql -U your_db_user -h your_db_host -d your_db_name -c "SELECT id, delegated_prefix_1, delegated_prefix_2, delegated_prefix_3 FROM tunnels LIMIT 5;"
```

### 6. Update Firewall Rules

If you have firewall rules that filter based on delegated prefixes, update them to include the new third prefix range:

```bash
# Example iptables rule for the new prefix range
iptables -A FORWARD -p ipv6 -s 2a06:xxxx:xxxx::/48 -j ACCEPT
```

### 7. Update Monitoring and Logging

Update your monitoring and logging systems to track the new third prefix:

1. Add the third prefix to your prefix utilization reports
2. Update any dashboards that display prefix information
3. Ensure logs capture events related to the third prefix

## Rollback Plan

If issues occur during migration, follow these steps to roll back:

1. Stop the TunnelBroker service:
   ```bash
   systemctl stop tunnelbroker
   ```

2. Revert the database changes:
   ```bash
   psql -U your_db_user -h your_db_host -d your_db_name -c "ALTER TABLE tunnels DROP COLUMN delegated_prefix_3;"
   ```

3. Restore the previous configuration:
   ```bash
   cp /path/to/backup/config.yaml /etc/tunnelbroker/config.yaml
   ```

4. Deploy the previous version of the software:
   ```bash
   cp /path/to/backup/tunnelbroker /usr/local/bin/tunnelbroker
   ```

5. Start the service:
   ```bash
   systemctl start tunnelbroker
   ```

## Support

If you encounter any issues during migration, please contact support at support@example.com or open an issue on our GitHub repository.
