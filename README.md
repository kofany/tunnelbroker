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
- Systemd service integration with installation scripts
- Advanced security features (since v0.0.6):
  - Traffic rate limiting (50 Mbps for both upload and download)
  - Email port blocking (SMTP, POP3, IMAP)
  - DDoS protection with rate limiting
  - SYN flood protection
  - ICMPv6 flood protection
  - Automatic security rules application for new tunnels
  - Fragment packet protection
  - Port scanning protection
  - New connection rate limiting

## Requirements

- Go 1.23 or newer
- PostgreSQL (Supabase)
- Linux with sit and gre modules
- Root privileges for tunnel management
- Traffic Control (tc) for bandwidth management
- iptables for security rules

## Installation

### Automatic Installation

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

4. Run the installation script:
```bash
sudo ./scripts/systemd/install.sh
```

### Manual Installation

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

### Uninstallation

To remove the service and all its components:
```bash
sudo ./scripts/systemd/uninstall.sh
```

## Security Features (v0.0.6+)

### Traffic Control
- Upload and download limited to 50 Mbps per tunnel
- Implemented using tc HTB qdisc
- Separate ingress and egress traffic shaping

### DDoS Protection
- SYN flood protection (1 packet/second limit)
- ICMPv6 flood protection (1 packet/second limit)
- General DDoS protection (10000 packets/second limit)
- New connection rate limiting (50 connections/second)

### Port Security
- Blocked email ports:
  - SMTP: 25, 465, 587, 2525
  - POP3: 110, 995
  - IMAP: 143, 993

### Additional Protection
- Fragment packet blocking
- TCP flag manipulation protection
- Port scanning protection
- Automatic security rules application for new tunnels

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

## Current Status

Version: v0.0.6
- Added automatic security rules application for new tunnels
- Added traffic rate limiting (50 Mbps up/down)
- Added DDoS protection
- Added email port blocking
- Added fragment packet protection
- Added port scanning protection
- Added connection rate limiting
- All messages translated to English
- Proper IPv6 address formatting
- Full API implementation
- Database integration with Supabase
- Automatic tunnel configuration generation
- Support for both SIT and GRE tunnels

## License

MIT
