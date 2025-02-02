# TunnelBroker API Documentation

## Podstawowe informacje

- Base URL: `http://localhost:9090/api/v1`
- Format danych: JSON
- Autentykacja: Bearer Token (w trybie produkcyjnym)

## Endpointy

### Tunele

#### Tworzenie tunelu
```http
POST /tunnels
```

**Request Body:**
```json
{
    "type": "sit",        // "sit" lub "gre"
    "user_id": 123,       // ID użytkownika z panelu
    "client_ipv4": "141.11.62.211",
    "server_ipv4": "192.67.35.38"
}
```

**Response:** (201 Created)
```json
{
    "tunnel_id": "tun123",
    "server_ipv4": "192.67.35.38",
    "client_ipv4": "141.11.62.211",
    "endpoint_local": "2a05:dfc3:ff:ffff:1::2/64",
    "endpoint_remote": "2a05:dfc3:ff:ffff:1::1/64",
    "delegated_prefixes": [
        "2a05:dfc3:ff:9875:237e::/64",
        "2a05:10:e2c1:dd71::/64"
    ],
    "routes": [
        {
            "prefix": "2a05:dfc3:ff:9875:237e::/64",
            "via": "2a05:dfc3:ff:ffff:1::1/64"
        },
        {
            "prefix": "2a05:10:e2c1:dd71::/64",
            "via": "2a05:dfc3:ff:ffff:1::1/64"
        }
    ]
}
```

#### Pobieranie komend konfiguracyjnych
```http
GET /tunnels/{id}/commands
```

**Response:** (200 OK)
```json
{
    "server": {
        "module_load": "modprobe sit",
        "setup": [
            "ip tunnel add tun123 mode sit remote 141.11.62.211 local 192.67.35.38 ttl 64",
            "ip link set tun123 up mtu 1480",
            "ip -6 addr add 2a05:dfc3:ff:ffff:1::1/64 dev tun123 nodad",
            "ip -6 route add 2a05:dfc3:ff:9875:237e::/64 dev tun123 metric 1",
            "ip -6 route add 2a05:10:e2c1:dd71::/64 dev tun123 metric 1"
        ],
        "teardown": [
            "ip tunnel del tun123"
        ]
    },
    "client": {
        "module_load": "modprobe sit",
        "setup": [
            "ip tunnel add tun123 mode sit remote 192.67.35.38 local 141.11.62.211 ttl 64",
            "ip link set tun123 up mtu 1480",
            "ip -6 addr add 2a05:dfc3:ff:ffff:1::2/64 dev tun123 nodad"
        ],
        "default_route": "ip -6 route add ::/0 via 2a05:dfc3:ff:ffff:1::1 dev tun123 metric 1",
        "selective_routes": [
            "ip -6 route add 2a05:dfc3:ff:9875:237e::/64 dev tun123 metric 1",
            "ip -6 route add 2a05:10:e2c1:dd71::/64 dev tun123 metric 1"
        ]
    },
    "info": {
        "tunnel_id": "tun123",
        "type": "sit",
        "server_ipv4": "192.67.35.38",
        "client_ipv4": "141.11.62.211",
        "ula_server": "2a05:dfc3:ff:ffff:1::1/64",
        "ula_client": "2a05:dfc3:ff:ffff:1::2/64",
        "prefix1": "2a05:dfc3:ff:9875:237e::/64",
        "prefix2": "2a05:10:e2c1:dd71::/64"
    }
}
```

#### Lista tuneli
```http
GET /tunnels
```

**Response:** (200 OK)
```json
[
    {
        "id": "tun123",
        "user_id": 123,
        "type": "sit",
        "client_ipv4": "141.11.62.211",
        "server_ipv4": "192.67.35.38",
        "status": "active",
        "endpoint_prefix": "2a05:dfc3:ff:ffff:1::1/64",
        "prefix1": "2a05:dfc3:ff:9875:237e::/64",
        "prefix2": "2a05:10:e2c1:dd71::/64",
        "created_at": "2024-01-29T11:52:02Z",
        "modified_at": "2024-01-29T11:52:02Z"
    }
]
```

#### Szczegóły tunelu
```http
GET /tunnels/{id}
```

**Response:** (200 OK)
```json
{
    "id": "tun123",
    "user_id": 123,
    "type": "sit",
    "client_ipv4": "141.11.62.211",
    "server_ipv4": "192.67.35.38",
    "status": "active",
    "endpoint_prefix": "2a05:dfc3:ff:ffff:1::1/64",
    "prefix1": "2a05:dfc3:ff:9875:237e::/64",
    "prefix2": "2a05:10:e2c1:dd71::/64",
    "created_at": "2024-01-29T11:52:02Z",
    "modified_at": "2024-01-29T11:52:02Z"
}
```

#### Usuwanie tunelu
```http
DELETE /tunnels/{id}
```

**Response:** (204 No Content)

#### Zawieszanie tunelu
```http
PUT /tunnels/{id}/suspend
```

**Response:** (200 OK)

#### Aktywacja tunelu
```http
PUT /tunnels/{id}/activate
```

**Response:** (200 OK)

### Prefiksy

#### Lista prefiksów
```http
GET /prefixes
```

**Response:** (200 OK)
```json
[
    {
        "prefix": "2a05:dfc3:ff00::/40",
        "is_endpoint": false,
        "description": "First pool",
        "created_at": "2024-01-29T11:52:02Z"
    }
]
```

#### Dodawanie prefiksu
```http
POST /prefixes
```

**Request Body:**
```json
{
    "prefix": "2a05:dfc3:ff00::/40",
    "description": "First pool"
}
```

**Response:** (201 Created)

### System

#### Status systemu
```http
GET /system/status
```

**Response:** (200 OK)
```json
{
    "total_tunnels": 1,
    "active_tunnels": 1,
    "total_prefixes": 4,
    "status": "operational",
    "version": "1.0.0"
}
```

## Kody błędów

- 400 Bad Request - Nieprawidłowe dane wejściowe
- 401 Unauthorized - Brak autoryzacji
- 403 Forbidden - Brak uprawnień
- 404 Not Found - Zasób nie znaleziony
- 409 Conflict - Konflikt (np. tunel już istnieje)
- 500 Internal Server Error - Błąd serwera

## Uwagi

1. Nazewnictwo tuneli:
   - Tunele SIT: `tunX` gdzie X to user_id
   - Tunele GRE: `tun-greX` gdzie X to user_id

2. Prefiksy IPv6:
   - Każdy tunel otrzymuje dwa prefiksy /64
   - Prefiksy są przydzielane z odpowiedniej pary pul
   - Endpointy używają adresów ULA (fde4:5a50:1114::/48)

3. Bezpieczeństwo:
   - W trybie produkcyjnym wymagana jest autoryzacja OAuth2
   - Token dostępu należy przekazywać w nagłówku Authorization
   - Dostęp tylko dla użytkowników z odpowiednimi uprawnieniami 