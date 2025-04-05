# Third IPv6 Prefix Implementation

This document describes the implementation of a third IPv6 prefix delegation for each tunnel in the TunnelBroker system.

## Overview

The TunnelBroker system has been updated to delegate three /64 prefixes per tunnel:
1. First prefix: from primary /44 range (e.g., 2a05:dfc1:3c00::/44)
2. Second prefix: from secondary /44 range (e.g., 2a12:bec0:2c00::/44)
3. New third prefix: from dedicated /48 range (e.g., 2a06:1234:5600::/48)

## Implementation Details

### Configuration

A new configuration field has been added to specify the dedicated /48 prefix for the third prefix delegation:

```yaml
prefixes:
  para1:
    primary: "2a05:xxxx:xxxx::/44"
    secondary: "2a12:xxxx:xxxx::/44"
  para2:
    primary: "2a05:xxxx:xxxx::/44"
    secondary: "2a05:xxxx:xxxx::/44"
  ula: "fde4:xxxx:xxxx::/48"
  third: "2a06:xxxx:xxxx::/48"  # New dedicated /48 prefix for third delegation
```

### Database Changes

The `tunnels` table has been updated with a new column:

```sql
ALTER TABLE public.tunnels ADD COLUMN delegated_prefix_3 text;
```

### Prefix Generation

A new function `generateThirdPrefix` has been implemented to generate the third prefix from the dedicated /48 range:

```go
// Format: {/48_prefix}:{user_id}::/64
func generateThirdPrefix(basePrefix string, userID string) (string, error) {
    // Implementation details
}
```

Unlike the first two prefixes which include a random hex digit, the third prefix uses a simpler format that directly embeds the user ID.

### Command Generation

The system now generates additional commands for both server and client to configure the third prefix:

#### Server Commands
```
ip -6 route add {delegated_prefix_3} dev {tunnel_id}
```

#### Client Commands
```
ip -6 addr add {delegated_prefix_3}1/64 dev {tunnel_id}
```

### Backward Compatibility

The implementation maintains backward compatibility with existing tunnels:
- For existing tunnels, the third prefix field will be NULL
- The command generation logic checks if the third prefix exists before adding related commands
- No changes to existing prefixes or addressing scheme

## Example Configuration

Before:
```
Tunnel (user_id="abcd"):
- ULA: fde4:5a50:1114:beef::1/64 (server)
- ULA: fde4:5a50:1114:beef::2/64 (client)
- Prefix1: 2a05:dfc1:3c0F:abcd::/64
- Prefix2: 2a12:bec0:2c0F:abcd::/64
```

After:
```
Tunnel (user_id="abcd"):
- ULA: fde4:5a50:1114:beef::1/64 (server)
- ULA: fde4:5a50:1114:beef::2/64 (client)
- Prefix1: 2a05:dfc1:3c0F:abcd::/64
- Prefix2: 2a12:bec0:2c0F:abcd::/64
- Prefix3: 2a06:1234:5600:abcd::/64
```

## Migration

To migrate existing tunnels, run the database migration script:

```sql
ALTER TABLE public.tunnels ADD COLUMN delegated_prefix_3 text;
```

Note that existing tunnels will have NULL for the third prefix until they are recreated or manually updated.
