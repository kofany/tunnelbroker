# TunnelBroker

Serwer automatyzujący tworzenie i zarządzanie tunelami IPv6 (SIT/GRE).

## Funkcjonalności

- Zarządzanie prefiksami IPv6
- Tworzenie i zarządzanie tunelami SIT i GRE
- Automatyczna alokacja prefiksów /64
- Generowanie konfiguracji dla klientów
- API REST do zarządzania systemem

## Wymagania

- Go 1.23 lub nowszy
- PostgreSQL (Supabase)
- Linux z modułami sit i gre
- Uprawnienia root do zarządzania tunelami

## Instalacja

1. Sklonuj repozytorium:
```bash
git clone https://github.com/kofany/tunnelbroker.git
cd tunnelbroker
```

2. Skopiuj i dostosuj pliki konfiguracyjne:
```bash
cp .env.example .env
cp cmd/config/config.example.yaml cmd/config/config.yaml
```

3. Edytuj pliki konfiguracyjne:
- `.env` - ustaw dane dostępowe do bazy danych
- `cmd/config/config.yaml` - skonfiguruj prefixy IPv6, adres serwera i klucz API

4. Zainstaluj zależności:
```bash
go mod download
```

5. Uruchom serwer:
```bash
GIN_MODE=release go run cmd/tunnelbroker/main.go
```

## Konfiguracja

Domyślnie serwer nasłuchuje na porcie 8080. Konfiguracja przechowywana jest w `/etc/tunnelbroker/`.

## API

Serwer nasłuchuje domyślnie na `127.0.0.1:8080`. Wszystkie endpointy wymagają nagłówka `X-API-Key`.

### Endpointy

1. Tworzenie tunelu:
```bash
POST /api/v1/tunnels
{
    "type": "sit|gre",
    "user_id": "hex4",
    "client_ipv4": "x.x.x.x"
}
```

2. Aktualizacja IP klienta:
```bash
PATCH /api/v1/tunnels/{tunnel_id}/ip
{
    "client_ipv4": "x.x.x.x"
}
```

3. Usuwanie tunelu:
```bash
DELETE /api/v1/tunnels/{tunnel_id}
```

## Bezpieczeństwo

- API dostępne tylko lokalnie (127.0.0.1)
- Wymagany klucz API dla wszystkich żądań
- Wrażliwe dane przechowywane w plikach konfiguracyjnych (nie w repozytorium)
- Limit 2 aktywnych tuneli na użytkownika

## Licencja

MIT
