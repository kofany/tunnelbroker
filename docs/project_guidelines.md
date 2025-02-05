# TunnelBroker - Wytyczne Implementacyjne

## 1. Założenia Główne

### 1.1 Prefixy IPv6
- 4 główne klasy /44:
  - 2a05:dfc1:3c00::/44
  - 2a12:bec0:2c00::/44
  - 2a05:1083:0000::/44
  - 2a05:dfc3:ff00::/44
- ULA prefix:
  - fde4:5a50:1114::/48 (jeden wspólny dla wszystkich tuneli)

### 1.2 Zasady Przydzielania Prefixów
- Każdy użytkownik może mieć maksymalnie 2 tunele
- Pierwszy tunel -> prefixy z pierwszej pary
- Drugi tunel -> prefixy z drugiej pary
- Jeden tunel = dwa prefixy /64 z tej samej pary
- UserID (4 znaki hex) jest używany w 4 bloku adresu
- Losowy znak hex w 3 bloku dla rozproszenia ruchu

### 1.3 Adresacja ULA
- Jeden wspólny prefix ULA: fde4:5a50:1114::/48
- Przydzielanie sekwencyjne /64 dla każdego tunelu
- Format: fde4:5a50:1114:XXXX::/64 (gdzie XXXX to kolejny numer)
- Dla każdego tunelu:
  - Serwer: fde4:5a50:1114:XXXX::1/64
  - Klient: fde4:5a50:1114:XXXX::2/64

## 2. Architektura Systemu

### 2.1 Backend (Go)
- Serwis systemowy w Debian 12
- REST API dla frontendu
- Zarządzanie tunelami i prefixami
- Wykonywanie komend systemowych
- Logowanie operacji

### 2.2 Frontend
- Interfejs webowy
- Maksymalnie 2 tunele na użytkownika
- Wyświetlanie konfiguracji dla klienta
- Monitoring stanu tuneli

### 2.3 Baza Danych (Supabase)
- Wspólna baza dla frontend i backend
- Wbudowana autentykacja
- Real-time aktualizacje
- Row Level Security

## 3. Schemat Bazy Danych

### 3.1 Tabela users
```sql
create table public.users (
  id uuid references auth.users primary key,
  hex_id char(4) unique not null,  -- 4-znakowy hex ID
  created_at timestamptz default now()
);
```

### 3.2 Tabela tunnels
```sql
create table public.tunnels (
  id text primary key,
  user_id uuid references public.users,
  type text check (type in ('sit', 'gre')),
  pair_number integer check (pair_number in (1, 2)),
  random_third_block char(1),
  server_ipv4 text,
  client_ipv4 text,
  ula_number bigint,
  status text default 'active',
  created_at timestamptz default now(),
  updated_at timestamptz default now(),
  unique(user_id, pair_number)
);
```

## 4. API Endpoints

### 4.1 Tworzenie Tunelu
```http
POST /api/v1/tunnels
{
    "type": "sit",        // sit lub gre
    "user_id": "beef",    // 4 znaki hex
    "client_ipv4": "141.11.62.211"
}
```

### 4.2 Odpowiedź
```json
{
    "tunnel_id": "tun-beef-1",
    "server_ipv4": "192.67.35.38",
    "client_ipv4": "141.11.62.211",
    "endpoint_local": "fde4:5a50:1114:0001::2/64",
    "endpoint_remote": "fde4:5a50:1114:0001::1/64",
    "delegated_prefixes": [
        "2a05:dfc1:3ca4:beef::/64",
        "2a12:bec0:2ca4:beef::/64"
    ]
}
```

## 5. Przykłady Adresacji

### 5.1 Pierwszy Tunel Użytkownika (ID: beef)
```
Para 1:
- Prefix 1: 2a05:dfc1:3c04:beef::/64 (losowe '4' w ostatnim polu 3 bloku)
- Prefix 2: 2a12:bec0:2c04:beef::/64 (to samo losowe '4' w ostatnim polu 3 bloku)
ULA: fde4:5a50:1114:0001::/64
```

### 5.2 Drugi Tunel Tego Samego Użytkownika
```
Para 2:
- Prefix 1: 2a05:1083:0007:beef::/64 (losowe '7' w ostatnim polu 3 bloku)
- Prefix 2: 2a05:dfc3:ff07:beef::/64 (to samo losowe '7' w ostatnim polu 3 bloku)
ULA: fde4:5a50:1114:0002::/64
```

Uwaga: Dla prefixów /44 losujemy tylko ostatnie pole trzeciego bloku (jeden znak hex), 
co daje nam 16 możliwych wartości (0-f) dla każdej podsieci. To pole musi być takie samo 
dla obu prefixów w parze przydzielonej do danego tunelu.

## 6. Bezpieczeństwo

### 6.1 Autentykacja
- Supabase Auth dla użytkowników
- API key dla backendu

### 6.2 Autoryzacja
- Row Level Security w bazie danych
- Maksymalnie 2 tunele na użytkownika
- Walidacja danych wejściowych

## 7. Monitoring

### 7.1 Frontend
- Status tuneli
- Wykorzystanie prefixów
- Logi operacji

### 7.2 Backend
- Stan interfejsów
- Routing
- Metryki systemowe 