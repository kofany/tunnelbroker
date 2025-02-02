# TunnelBroker

TunnelBroker to serwer do zarządzania tunelami IPv6 poprzez API REST.

## Funkcjonalności

- Zarządzanie prefiksami IPv6
- Tworzenie i zarządzanie tunelami SIT i GRE
- Automatyczna alokacja prefiksów /64
- Generowanie konfiguracji dla klientów
- API REST do zarządzania systemem

## Wymagania

- Go 1.21 lub nowszy
- Uprawnienia root do tworzenia tuneli
- System Linux z obsługą tuneli SIT i GRE
- Moduły jądra: sit, gre

## Instalacja

```bash
# Klonowanie repozytorium
git clone https://github.com/kofany/tunnelbroker.git
cd tunnelbroker

# Instalacja zależności
go mod download

# Budowanie
go build -o tunnelbroker cmd/tunnelbroker/main.go

# Upewnij się, że moduły jądra są załadowane
modprobe sit
modprobe gre
```

## Konfiguracja

Domyślnie serwer nasłuchuje na porcie 8080. Konfiguracja przechowywana jest w `/etc/tunnelbroker/`.

## API

### Tunele

- `POST /api/v1/tunnels` - Tworzenie nowego tunelu
- `GET /api/v1/tunnels` - Lista tuneli
- `GET /api/v1/tunnels/{id}` - Szczegóły tunelu
- `DELETE /api/v1/tunnels/{id}` - Usunięcie tunelu
- `PUT /api/v1/tunnels/{id}/suspend` - Zawieszenie tunelu
- `PUT /api/v1/tunnels/{id}/activate` - Aktywacja tunelu

### Prefiksy

- `POST /api/v1/prefixes` - Dodanie prefiksu
- `GET /api/v1/prefixes` - Lista prefiksów
- `PUT /api/v1/prefixes/endpoint` - Ustawienie prefiksu końcowego

### System

- `GET /api/v1/system/status` - Status systemu
- `GET /api/v1/system/config/{id}` - Konfiguracja klienta

## Przykłady użycia

### Tworzenie tunelu SIT

```bash
curl -X POST http://localhost:8080/api/v1/tunnels \
  -H "Content-Type: application/json" \
  -d '{
    "type": "sit",
    "client_nickname": "example",
    "client_ipv4": "192.0.2.1",
    "server_ipv4": "192.0.2.2"
  }'
```

### Tworzenie tunelu GRE

```bash
curl -X POST http://localhost:8080/api/v1/tunnels \
  -H "Content-Type: application/json" \
  -d '{
    "type": "gre",
    "client_nickname": "example-gre",
    "client_ipv4": "192.0.2.3",
    "server_ipv4": "192.0.2.4"
  }'
```

### Dodanie prefiksu

```bash
curl -X POST http://localhost:8080/api/v1/prefixes \
  -H "Content-Type: application/json" \
  -d '{
    "prefix": "2001:db8::/32",
    "description": "Main prefix pool"
  }'
```

## Licencja

MIT
