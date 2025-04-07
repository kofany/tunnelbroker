# TunnelBroker API: User Tunnels Endpoint

## Overview

This endpoint allows retrieving all tunnels associated with a specific user ID. It returns the same response format as other tunnel endpoints, maintaining consistency across the API.

## Endpoint Specification

### Request

```http
GET /api/v1/tunnels/user/{user_id}
```

### Headers

| Header      | Value                                                | Required |
|-------------|------------------------------------------------------|----------|
| X-API-Key   | tb-sHj8Xp2qLkRtY7vZnM9wBcDfG3hJ6kL5mN8pQ7rS4tU2vW5xY9zA | Yes      |

### Path Parameters

| Parameter | Type   | Description                                 |
|-----------|--------|---------------------------------------------|
| user_id   | string | User ID (must be exactly 4 characters long) |

### Response Format

The endpoint returns an array of tunnel objects, each containing:

- `tunnel`: Object containing tunnel details
  - `id`: Tunnel identifier (format: "tun-{user_id}-{number}")
  - `user_id`: User identifier
  - `type`: Tunnel type ("sit" or "gre")
  - `status`: Tunnel status ("active" or "inactive")
  - `server_ipv4`: Server IPv4 address
  - `client_ipv4`: Client IPv4 address
  - `endpoint_local`: Local IPv6 endpoint
  - `endpoint_remote`: Remote IPv6 endpoint
  - `delegated_prefix_1`: First delegated IPv6 prefix
  - `delegated_prefix_2`: Second delegated IPv6 prefix
  - `delegated_prefix_3`: Third delegated IPv6 prefix
  - `created_at`: Timestamp when the tunnel was created
- `commands`: Object containing configuration commands
  - `server`: Array of commands to configure the server side
  - `client`: Array of commands to configure the client side

### Response Codes

| Status Code | Description                                                |
|-------------|------------------------------------------------------------|
| 200         | Success (returns array of tunnels, may be empty)           |
| 400         | Bad Request (invalid user_id format)                       |
| 401         | Unauthorized (invalid API key)                             |
| 500         | Internal Server Error                                      |

## Examples

### Example 1: Successful Request

#### Request

```bash
curl -X GET http://127.0.0.1:8080/api/v1/tunnels/user/2034 \
  -H "X-API-Key: tb-sHj8Xp2qLkRtY7vZnM9wBcDfG3hJ6kL5mN8pQ7rS4tU2vW5xY9zA"
```

#### Response (200 OK)

```json
[
  {
    "tunnel": {
      "id": "tun-2034-1",
      "user_id": "2034",
      "type": "sit",
      "status": "active",
      "server_ipv4": "185.243.218.164",
      "client_ipv4": "188.68.232.86",
      "endpoint_local": "fde4:5a50:1114:9f6b::1/64",
      "endpoint_remote": "fde4:5a50:1114:9f6b::2/64",
      "delegated_prefix_1": "2a05:1083:bef5:2034::/64",
      "delegated_prefix_2": "2a12:bec0:2c4:2034::/64",
      "delegated_prefix_3": "2a03:94e0:2496:2034::/64",
      "created_at": "2025-04-06T00:48:45.260791+02:00"
    },
    "commands": {
      "server": [
        "ip tunnel add tun-2034-1 mode sit local 185.243.218.164 remote 188.68.232.86 ttl 255",
        "ip link set tun-2034-1 up",
        "ip -6 addr add fde4:5a50:1114:9f6b::1/64 dev tun-2034-1",
        "ip -6 route add 2a05:1083:bef5:2034::/64 dev tun-2034-1",
        "ip -6 route add 2a12:bec0:2c4:2034::/64 dev tun-2034-1",
        "ip -6 route add 2a03:94e0:2496:2034::/64 dev tun-2034-1"
      ],
      "client": [
        "ip tunnel add tun-2034-1 mode sit local 188.68.232.86 remote 185.243.218.164 ttl 255",
        "ip link set tun-2034-1 up",
        "ip -6 addr add fde4:5a50:1114:9f6b::2/64 dev tun-2034-1",
        "ip -6 addr add 2a05:1083:bef5:2034::1/64 dev tun-2034-1",
        "ip -6 addr add 2a12:bec0:2c4:2034::1/64 dev tun-2034-1",
        "ip -6 route add ::/0 via fde4:5a50:1114:9f6b::1 dev tun-2034-1",
        "ip -6 addr add 2a03:94e0:2496:2034::1/64 dev tun-2034-1"
      ]
    }
  }
]
```

### Example 2: User with No Tunnels

#### Request

```bash
curl -X GET http://127.0.0.1:8080/api/v1/tunnels/user/9999 \
  -H "X-API-Key: tb-sHj8Xp2qLkRtY7vZnM9wBcDfG3hJ6kL5mN8pQ7rS4tU2vW5xY9zA"
```

#### Response (200 OK)

```json
[]
```

### Example 3: Invalid User ID Format

#### Request

```bash
curl -X GET http://127.0.0.1:8080/api/v1/tunnels/user/123 \
  -H "X-API-Key: tb-sHj8Xp2qLkRtY7vZnM9wBcDfG3hJ6kL5mN8pQ7rS4tU2vW5xY9zA"
```

#### Response (400 Bad Request)

```json
{
  "error": "Invalid user_id format. Must be 4 characters."
}
```

### Example 4: Invalid API Key

#### Request

```bash
curl -X GET http://127.0.0.1:8080/api/v1/tunnels/user/2034 \
  -H "X-API-Key: invalid-key"
```

#### Response (401 Unauthorized)

```json
{
  "error": "Invalid API key"
}
```

## Error Handling

The endpoint handles the following error cases:

1. **Invalid User ID Format**: If the user_id is not exactly 4 characters long, returns a 400 Bad Request with an error message.
2. **Invalid API Key**: If the X-API-Key header is missing or invalid, returns a 401 Unauthorized with an error message.
3. **Internal Server Error**: If there's an error retrieving data from the database, returns a 500 Internal Server Error with an error message.

## Notes

- The endpoint returns an empty array (`[]`) when no tunnels are found for the specified user ID, rather than a 404 Not Found response.
- All tunnels are returned in the same format as the `/api/v1/tunnels/{tunnel_id}` endpoint, maintaining API consistency.
- The response includes both tunnel details and the commands needed to configure both server and client sides.
