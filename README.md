# TunnelBroker

A server application for automating IPv6 tunnel (SIT/GRE) creation and management.

## Features

- Automated IPv6 prefix management and delegation
- SIT and GRE tunnel creation and management
- Automatic /64 prefix allocation from /44 pools
- Client configuration generation with proper IPv6 formatting
- RESTful API with authentication
- Automatic ULA address generation for tunnel endpoints
- Support for dual prefix delegation (primary and secondary)
- User-based tunnel management with limits
- Systemd service integration

## Requirements

- Go 1.23 or newer
- PostgreSQL (Supabase)
- Linux with sit and gre modules
- Root privileges for tunnel management

## Installation

1. Clone the repository:
```bash
git clone https://github.com/kofany/tunnelbroker.git
cd tunnelbroker
```

2. Copy and customize configuration files:
```bash
cp .env.example .env
cp cmd/config/config.example.yaml cmd/config/config.yaml
```

3. Edit configuration files:
- `.env` - set database credentials
- `cmd/config/config.yaml` - configure IPv6 prefixes, server address and API key

4. Install dependencies:
```bash
go mod download
```

5. Build and install the service:
```bash
go build -o /usr/local/bin/tunnelbroker cmd/tunnelbroker/main.go
cp /etc/systemd/system/tunnelbroker.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable tunnelbroker
systemctl start tunnelbroker
```

## Configuration

The server listens on port 8080 by default. Configuration is stored in `/etc/tunnelbroker/`.

## API

The server listens on `127.0.0.1:8080` by default. All endpoints require the `X-API-Key` header.

### Endpoints

1. Create a tunnel:
```bash
POST /api/v1/tunnels
{
    "type": "sit|gre",
    "user_id": "hex4",
    "client_ipv4": "x.x.x.x"
}
```

2. Update client IP:
```bash
PATCH /api/v1/tunnels/{tunnel_id}/ip
{
    "client_ipv4": "x.x.x.x"
}
```

3. Delete tunnel:
```bash
DELETE /api/v1/tunnels/{tunnel_id}
```

4. List tunnels:
```bash
GET /api/v1/tunnels?user_id={user_id}
```

5. Get tunnel details:
```bash
GET /api/v1/tunnels/{tunnel_id}
```

## Security

- API only available locally (127.0.0.1)
- API key required for all requests
- Sensitive data stored in configuration files (not in repository)
- 2 active tunnels per user limit
- Proper IPv6 address formatting and validation
- Automatic ULA address generation for tunnel endpoints

## Database Schema

The application uses three main tables:
- `tunnels` - stores tunnel configurations
- `users` - manages user tunnel limits
- `prefixes` - tracks IPv6 prefix allocations

## Current Status

Version: v0.0.1
- Initial release with proper IPv6 address formatting
- Full API implementation
- Systemd service integration
- Database integration with Supabase
- Automatic tunnel configuration generation
- Support for both SIT and GRE tunnels

## License

MIT
