# WireGuard Tunneling Support - Frontend Integration Specification

## 📋 Executive Summary

Tunnel Broker API został rozszerzony o obsługę tuneli WireGuard (`type: "wg"`) obok istniejących typów SIT i GRE. Dodano automatyczne generowanie kluczy kryptograficznych, rozszerzono schemat bazy danych oraz zaktualizowano wszystkie endpointy API.

**Data implementacji:** 2024-11-12
**Wersja API:** Bez zmian (backward compatible)
**Breaking changes:** Brak

---

## 🎯 Co się zmieniło

### Nowe możliwości
- ✅ Tworzenie tuneli WireGuard (`type: "wg"`)
- ✅ Automatyczne generowanie par kluczy WireGuard (Curve25519)
- ✅ Przydzielanie portów nasłuchiwania (51820-51821)
- ✅ Pełna obsługa w tunnelrecovery (automatyczne odtwarzanie po restarcie)

### Niezmienione funkcjonalności
- ✅ Tunele SIT i GRE działają identycznie jak wcześniej
- ✅ Wszystkie istniejące endpointy zachowują kompatybilność wsteczną
- ✅ Limity użytkowników (max 2 tunele) bez zmian
- ✅ System delegacji prefixów IPv6 bez zmian

---

## 🔧 Zmiany w API

### 1. Nowe pola w obiekcie `Tunnel`

```typescript
interface Tunnel {
  // Istniejące pola (bez zmian)
  id: string;                    // "tun-abcd-1"
  user_id: string;               // "abcd"
  type: string;                  // "sit" | "gre" | "wg" ⬅️ NOWE: "wg"
  status: string;                // "active" | "suspended"
  server_ipv4: string;           // "1.2.3.4"
  client_ipv4: string;           // "5.6.7.8"
  endpoint_local: string;        // "fde4:8dba:82e1::1/64"
  endpoint_remote: string;       // "fde4:8dba:82e1::2/64"
  delegated_prefix_1: string;    // "2a05:xxxx:xxxx::/64"
  delegated_prefix_2: string;    // "2a12:xxxx:xxxx::/64"
  delegated_prefix_3: string;    // "2a06:xxxx:xxxx::/64"
  created_at: string;            // ISO 8601 timestamp

  // ⬇️ NOWE pola dla WireGuard (nullable dla sit/gre)
  server_private_key?: string;   // Base64 encoded key
  server_public_key?: string;    // Base64 encoded key
  client_private_key?: string;   // Base64 encoded key
  client_public_key?: string;    // Base64 encoded key
  listen_port?: number;          // 51820 | 51821
}
```

### 2. Aktualizacja validation rules

**Endpoint:** `POST /api/v1/tunnels`

**Request body:**
```json
{
  "type": "sit" | "gre" | "wg",  // ⬅️ ZMIANA: dodano "wg"
  "user_id": "abcd",             // 4 znaki hex (bez zmian)
  "client_ipv4": "1.2.3.4"       // Valid IPv4 (bez zmian)
}
```

**Validation:**
- `type`: Wymagane, jedno z: `"sit"`, `"gre"`, `"wg"`
- `user_id`: Wymagane, dokładnie 4 znaki (bez zmian)
- `client_ipv4`: Wymagane, valid IPv4 address (bez zmian)

---

## 📡 Response Examples

### Tworzenie tunelu WireGuard

**Request:**
```bash
POST /api/v1/tunnels
Content-Type: application/json
X-API-Key: your_api_key

{
  "type": "wg",
  "user_id": "abcd",
  "client_ipv4": "93.184.216.34"
}
```

**Response:** `200 OK`
```json
{
  "tunnel": {
    "id": "tun-abcd-1",
    "user_id": "abcd",
    "type": "wg",
    "status": "active",
    "server_ipv4": "203.0.113.1",
    "client_ipv4": "93.184.216.34",
    "endpoint_local": "fde4:8dba:82e1::a3c1/64",
    "endpoint_remote": "fde4:8dba:82e1::a3c2/64",
    "delegated_prefix_1": "2a05:d014:1c0a:ab00::/64",
    "delegated_prefix_2": "2a12:5940:d891:ab00::/64",
    "delegated_prefix_3": "2a06:e881:7400:abcd::/64",
    "created_at": "2024-11-12T20:15:30Z",

    "server_private_key": "YAnz...base64...==",
    "server_public_key": "xTIU...base64...==",
    "client_private_key": "cNv4...base64...==",
    "client_public_key": "HIgo...base64...==",
    "listen_port": 51820
  },
  "commands": {
    "server": [
      "ip link add dev tun-abcd-1 type wireguard",
      "ip -6 addr add fde4:8dba:82e1::a3c1/64 dev tun-abcd-1",
      "wg set tun-abcd-1 listen-port 51820 private-key <(echo YAnz...==) peer HIgo...== allowed-ips fde4:8dba:82e1::a3c2/64,2a05:d014:1c0a:ab00::/64,2a12:5940:d891:ab00::/64",
      "ip link set tun-abcd-1 up",
      "ip -6 route add 2a05:d014:1c0a:ab00::/64 dev tun-abcd-1",
      "ip -6 route add 2a12:5940:d891:ab00::/64 dev tun-abcd-1",
      "ip -6 route add 2a06:e881:7400:abcd::/64 dev tun-abcd-1"
    ],
    "client": [
      "ip link add dev tun-abcd-1 type wireguard",
      "ip -6 addr add fde4:8dba:82e1::a3c2/64 dev tun-abcd-1",
      "ip -6 addr add 2a05:d014:1c0a:ab00::1/64 dev tun-abcd-1",
      "ip -6 addr add 2a12:5940:d891:ab00::1/64 dev tun-abcd-1",
      "ip -6 addr add 2a06:e881:7400:abcd::1/64 dev tun-abcd-1",
      "wg set tun-abcd-1 private-key <(echo cNv4...==) peer xTIU...== endpoint 203.0.113.1:51820 allowed-ips ::/0",
      "ip link set tun-abcd-1 up",
      "ip -6 route add ::/0 dev tun-abcd-1"
    ]
  }
}
```

### Tworzenie tunelu SIT (bez zmian)

**Request:**
```bash
POST /api/v1/tunnels
Content-Type: application/json
X-API-Key: your_api_key

{
  "type": "sit",
  "user_id": "abcd",
  "client_ipv4": "93.184.216.34"
}
```

**Response:** `200 OK`
```json
{
  "tunnel": {
    "id": "tun-abcd-1",
    "user_id": "abcd",
    "type": "sit",
    "status": "active",
    "server_ipv4": "203.0.113.1",
    "client_ipv4": "93.184.216.34",
    "endpoint_local": "fde4:8dba:82e1::a3c1/64",
    "endpoint_remote": "fde4:8dba:82e1::a3c2/64",
    "delegated_prefix_1": "2a05:d014:1c0a:ab00::/64",
    "delegated_prefix_2": "2a12:5940:d891:ab00::/64",
    "delegated_prefix_3": "2a06:e881:7400:abcd::/64",
    "created_at": "2024-11-12T20:15:30Z"

    // Brak pól WireGuard dla typu sit/gre
  },
  "commands": {
    "server": [
      "ip tunnel add tun-abcd-1 mode sit local 203.0.113.1 remote 93.184.216.34 ttl 255",
      "ip link set tun-abcd-1 up",
      "ip -6 addr add fde4:8dba:82e1::a3c1/64 dev tun-abcd-1",
      "ip -6 route add 2a05:d014:1c0a:ab00::/64 dev tun-abcd-1",
      "ip -6 route add 2a12:5940:d891:ab00::/64 dev tun-abcd-1",
      "ip -6 route add 2a06:e881:7400:abcd::/64 dev tun-abcd-1"
    ],
    "client": [
      "ip tunnel add tun-abcd-1 mode sit local 93.184.216.34 remote 203.0.113.1 ttl 255",
      "ip link set tun-abcd-1 up",
      "ip -6 addr add fde4:8dba:82e1::a3c2/64 dev tun-abcd-1",
      "ip -6 addr add 2a05:d014:1c0a:ab00::1/64 dev tun-abcd-1",
      "ip -6 addr add 2a12:5940:d891:ab00::1/64 dev tun-abcd-1",
      "ip -6 addr add 2a06:e881:7400:abcd::1/64 dev tun-abcd-1",
      "ip -6 route add ::/0 via fde4:8dba:82e1::a3c1 dev tun-abcd-1"
    ]
  }
}
```

### Listowanie tuneli (GET /api/v1/tunnels)

**Response zawiera nowe pola tylko dla tuneli WireGuard:**

```json
[
  {
    "tunnel": {
      "id": "tun-abcd-1",
      "type": "wg",
      // ... wszystkie pola włącznie z kluczami WireGuard
      "server_private_key": "YAnz...==",
      "server_public_key": "xTIU...==",
      "client_private_key": "cNv4...==",
      "client_public_key": "HIgo...==",
      "listen_port": 51820
    },
    "commands": { /* ... */ }
  },
  {
    "tunnel": {
      "id": "tun-ef01-1",
      "type": "sit",
      // ... brak pól WireGuard (lub null/undefined)
    },
    "commands": { /* ... */ }
  }
]
```

---

## 🔐 Szczegóły kluczy WireGuard

### Format kluczy
- **Algorytm:** Curve25519 (EdDSA)
- **Encoding:** Base64
- **Długość:** 44 znaki base64 (32 bajty raw)
- **Przykład:** `YAnz5vxXKObVkSTPvRr+Of27Py6mjgzmYHAvDLBJLlE=`

### Pary kluczy
Każdy tunel WireGuard ma **dwie pary kluczy**:

1. **Klucze serwera:**
   - `server_private_key` - klucz prywatny serwera (NIGDY nie pokazuj w UI!)
   - `server_public_key` - klucz publiczny serwera (bezpieczny do pokazania)

2. **Klucze klienta:**
   - `client_private_key` - klucz prywatny klienta (dla klienta do konfiguracji)
   - `client_public_key` - klucz publiczny klienta (używany przez serwer)

### Bezpieczeństwo
⚠️ **WAŻNE dla UI:**
- Klucze prywatne (`*_private_key`) powinny być:
  - Pokazywane tylko raz po utworzeniu tunelu
  - Ukrywane domyślnie (np. za buttonem "Show key")
  - Kopiowalne do schowka
  - Możliwość pobrania jako plik konfiguracyjny

---

## 🎨 Zmiany wymagane w Frontend Dashboard

### 1. Formularz tworzenia tunelu

**Pole Type - aktualizacja:**

```tsx
// Przed:
<Select name="type">
  <Option value="sit">SIT Tunnel</Option>
  <Option value="gre">GRE Tunnel</Option>
</Select>

// Po:
<Select name="type">
  <Option value="sit">SIT Tunnel</Option>
  <Option value="gre">GRE Tunnel</Option>
  <Option value="wg">WireGuard Tunnel ⭐ NEW</Option>
</Select>
```

**Dodatkowe informacje dla użytkownika:**

```tsx
{type === 'wg' && (
  <InfoBox>
    WireGuard tunnel provides modern, secure VPN technology with
    automatic key generation. Better performance than SIT/GRE.
  </InfoBox>
)}
```

### 2. Lista tuneli - identyfikacja typu

**Dodaj ikonę/badge dla typu tunelu:**

```tsx
const TunnelTypeBadge = ({ type }) => {
  const badges = {
    sit: { label: 'SIT', color: 'blue', icon: '🔵' },
    gre: { label: 'GRE', color: 'green', icon: '🟢' },
    wg: { label: 'WireGuard', color: 'purple', icon: '🔐' }
  };

  return (
    <Badge color={badges[type].color}>
      {badges[type].icon} {badges[type].label}
    </Badge>
  );
};
```

### 3. Szczegóły tunelu - warunkowe wyświetlanie kluczy

```tsx
const TunnelDetails = ({ tunnel }) => {
  const isWireGuard = tunnel.type === 'wg';

  return (
    <div>
      {/* Wspólne pola dla wszystkich typów */}
      <Field label="ID" value={tunnel.id} />
      <Field label="Type" value={<TunnelTypeBadge type={tunnel.type} />} />
      <Field label="Status" value={tunnel.status} />
      <Field label="Client IPv4" value={tunnel.client_ipv4} />
      <Field label="Endpoint Local" value={tunnel.endpoint_local} />
      <Field label="Endpoint Remote" value={tunnel.endpoint_remote} />

      {/* Prefixes */}
      <Section title="Delegated IPv6 Prefixes">
        <Field label="Prefix 1" value={tunnel.delegated_prefix_1} />
        <Field label="Prefix 2" value={tunnel.delegated_prefix_2} />
        <Field label="Prefix 3" value={tunnel.delegated_prefix_3} />
      </Section>

      {/* ⬇️ NOWA SEKCJA: tylko dla WireGuard */}
      {isWireGuard && (
        <Section title="WireGuard Configuration">
          <Field label="Listen Port" value={tunnel.listen_port} />

          <KeyField
            label="Server Public Key"
            value={tunnel.server_public_key}
            sensitive={false}
          />

          <KeyField
            label="Server Private Key"
            value={tunnel.server_private_key}
            sensitive={true}
            warning="Keep this secret! Never share."
          />

          <KeyField
            label="Client Public Key"
            value={tunnel.client_public_key}
            sensitive={false}
          />

          <KeyField
            label="Client Private Key"
            value={tunnel.client_private_key}
            sensitive={true}
            warning="Provide this to the client securely."
          />
        </Section>
      )}
    </div>
  );
};
```

### 4. Komponent do wyświetlania kluczy

```tsx
const KeyField = ({ label, value, sensitive, warning }) => {
  const [visible, setVisible] = useState(!sensitive);
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="key-field">
      <label>{label}</label>
      {warning && <Warning>{warning}</Warning>}

      <div className="key-value">
        <code>
          {visible ? value : '••••••••••••••••••••••••••••••••••••••••'}
        </code>

        <ButtonGroup>
          {sensitive && (
            <Button onClick={() => setVisible(!visible)}>
              {visible ? '👁️ Hide' : '👁️‍🗨️ Show'}
            </Button>
          )}

          <Button onClick={handleCopy}>
            {copied ? '✅ Copied' : '📋 Copy'}
          </Button>
        </ButtonGroup>
      </div>
    </div>
  );
};
```

### 5. Generowanie pliku konfiguracyjnego dla klienta

**Dodaj przycisk "Download WireGuard Config":**

```tsx
const downloadWireGuardConfig = (tunnel) => {
  if (tunnel.type !== 'wg') return;

  const config = `[Interface]
PrivateKey = ${tunnel.client_private_key}
Address = ${tunnel.endpoint_remote}
Address = ${tunnel.delegated_prefix_1.replace('/64', '::1/64')}
Address = ${tunnel.delegated_prefix_2.replace('/64', '::1/64')}
Address = ${tunnel.delegated_prefix_3.replace('/64', '::1/64')}

[Peer]
PublicKey = ${tunnel.server_public_key}
Endpoint = ${tunnel.server_ipv4}:${tunnel.listen_port}
AllowedIPs = ::/0
PersistentKeepalive = 25
`;

  const blob = new Blob([config], { type: 'text/plain' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${tunnel.id}.conf`;
  a.click();
  URL.revokeObjectURL(url);
};
```

### 6. Komponent Commands - bez zmian struktury

Komponent wyświetlający `commands.server` i `commands.client` **nie wymaga zmian** - struktura pozostaje taka sama, tylko same komendy są inne dla WireGuard.

```tsx
// Ten kod działa już dla wszystkich typów tuneli
const CommandsSection = ({ commands }) => (
  <div>
    <CodeBlock title="Server Commands" commands={commands.server} />
    <CodeBlock title="Client Commands" commands={commands.client} />
  </div>
);
```

---

## 🔄 Flow tworzenia tunelu WireGuard

```
┌─────────────────┐
│  User Dashboard │
│  (Frontend)     │
└────────┬────────┘
         │
         │ 1. POST /api/v1/tunnels
         │    { type: "wg", user_id: "abcd", client_ipv4: "..." }
         │
         ▼
┌─────────────────┐
│   TunnelBroker  │
│   API Server    │
└────────┬────────┘
         │
         │ 2. Generate WireGuard keys (Curve25519)
         │    - server_private_key / server_public_key
         │    - client_private_key / client_public_key
         │
         │ 3. Generate IPv6 prefixes (bez zmian)
         │    - delegated_prefix_1, _2, _3
         │
         │ 4. Assign listen port (51820 or 51821)
         │
         │ 5. Save to database
         │    INSERT INTO tunnels (... + WireGuard fields)
         │
         │ 6. Execute system commands
         │    ip link add dev tun-abcd-1 type wireguard
         │    wg set tun-abcd-1 ...
         │    ip link set tun-abcd-1 up
         │
         │ 7. Apply security rules
         │    /etc/tunnelbroker/scripts/tunnel_security.sh
         │
         ▼
┌─────────────────┐
│   Response      │
│   200 OK        │
└────────┬────────┘
         │
         │ tunnel object + commands
         │
         ▼
┌─────────────────┐
│  User Dashboard │
│  Shows:         │
│  - Tunnel info  │
│  - WireGuard    │
│    keys         │
│  - Config file  │
│    download     │
└─────────────────┘
```

---

## ⚙️ Różnice między typami tuneli

| Feature | SIT | GRE | WireGuard |
|---------|-----|-----|-----------|
| **Typ protokołu** | IPv6-in-IPv4 | Generic Routing | Modern VPN |
| **Komendy tworzenia** | `ip tunnel add mode sit` | `ip tunnel add mode gre` | `ip link add type wireguard` |
| **Komendy usuwania** | `ip tunnel del` | `ip tunnel del` | `ip link del` |
| **Enkrypcja** | ❌ Brak | ❌ Brak | ✅ Curve25519 |
| **Klucze** | ❌ Nie wymagane | ❌ Nie wymagane | ✅ Generowane automatycznie |
| **Port nasłuchiwania** | ❌ N/A | ❌ N/A | ✅ 51820-51821 |
| **Performance** | Dobra | Dobra | Najlepsza |
| **Bezpieczeństwo** | Niezaszyfrowane | Niezaszyfrowane | Zaszyfrowane |
| **Pola w bazie** | Podstawowe | Podstawowe | Podstawowe + 5 pól WG |

---

## 🗄️ Zmiany w bazie danych

### Nowe kolumny w tabeli `tunnels`

```sql
-- Dodane kolumny (nullable dla sit/gre)
ALTER TABLE public.tunnels
ADD COLUMN server_private_key TEXT,
ADD COLUMN server_public_key TEXT,
ADD COLUMN client_private_key TEXT,
ADD COLUMN client_public_key TEXT,
ADD COLUMN listen_port INTEGER;

-- Zaktualizowany constraint
ALTER TABLE public.tunnels DROP CONSTRAINT tunnels_type_check;
ALTER TABLE public.tunnels
ADD CONSTRAINT tunnels_type_check
CHECK (type = ANY (ARRAY['sit'::text, 'gre'::text, 'wg'::text]));
```

### Wartości dla różnych typów

**Dla tuneli SIT/GRE:**
```sql
server_private_key: NULL
server_public_key: NULL
client_private_key: NULL
client_public_key: NULL
listen_port: NULL
```

**Dla tuneli WireGuard:**
```sql
server_private_key: 'YAnz5vxXKObVkSTPvRr+Of27Py6mjgzmYHAvDLBJLlE='
server_public_key: 'xTIU3toiKlay9M8lz/C/iFSB7cMZ5PEX8Uv9y0LgSjA='
client_private_key: 'cNv4qhWW4TjhJNxqIc5t7D7tCtLi/GUgYhL7WCvXaHI='
client_public_key: 'HIgo3vWMjKBQZr2KDf4z8pOmH4luLMaGPpz5lqBxBWo='
listen_port: 51820
```

---

## 🎯 Checklist dla Frontend Developera

### Must Have (Wymagane)

- [ ] Dodać `"wg"` do opcji w dropdown typu tunelu
- [ ] Obsłużyć nowe pola w TypeScript interfaces
- [ ] Warunkowo wyświetlać sekcję "WireGuard Configuration"
- [ ] Implementować komponent do pokazywania/ukrywania kluczy prywatnych
- [ ] Dodać funkcję kopiowania kluczy do schowka
- [ ] Wyświetlać badge/ikonę typu tunelu w liście
- [ ] Zaktualizować TypeScript types dla `Tunnel` interface

### Should Have (Zalecane)

- [ ] Dodać przycisk "Download WireGuard Config"
- [ ] Pokazać tooltip z informacją o WireGuard przy wyborze typu
- [ ] Dodać sekcję pomocy "How to use WireGuard tunnel"
- [ ] Wyświetlać ostrzeżenia bezpieczeństwa przy kluczach prywatnych
- [ ] Zaimplementować modal potwierdzenia przed pokazaniem klucza prywatnego

### Nice to Have (Opcjonalne)

- [ ] QR code z konfiguracją WireGuard (dla aplikacji mobilnych)
- [ ] Walidacja formatowania kluczy WireGuard
- [ ] Statystyki porównawcze typów tuneli
- [ ] Auto-hide klucza prywatnego po 10 sekundach
- [ ] Eksport konfiguracji w różnych formatach (JSON, YAML, WireGuard .conf)

---

## 🧪 Przykładowe dane testowe

### Tunel WireGuard (pełny przykład)

```json
{
  "tunnel": {
    "id": "tun-test-1",
    "user_id": "test",
    "type": "wg",
    "status": "active",
    "server_ipv4": "203.0.113.1",
    "client_ipv4": "93.184.216.34",
    "endpoint_local": "fde4:8dba:82e1::1234/64",
    "endpoint_remote": "fde4:8dba:82e1::5678/64",
    "delegated_prefix_1": "2a05:d014:1c0a:ab00::/64",
    "delegated_prefix_2": "2a12:5940:d891:ab00::/64",
    "delegated_prefix_3": "2a06:e881:7400:test::/64",
    "created_at": "2024-11-12T20:00:00Z",
    "server_private_key": "YAnz5vxXKObVkSTPvRr+Of27Py6mjgzmYHAvDLBJLlE=",
    "server_public_key": "xTIU3toiKlay9M8lz/C/iFSB7cMZ5PEX8Uv9y0LgSjA=",
    "client_private_key": "cNv4qhWW4TjhJNxqIc5t7D7tCtLi/GUgYhL7WCvXaHI=",
    "client_public_key": "HIgo3vWMjKBQZr2KDf4z8pOmH4luLMaGPpz5lqBxBWo=",
    "listen_port": 51820
  }
}
```

### Tunel SIT (dla porównania)

```json
{
  "tunnel": {
    "id": "tun-test-2",
    "user_id": "test",
    "type": "sit",
    "status": "active",
    "server_ipv4": "203.0.113.1",
    "client_ipv4": "93.184.216.35",
    "endpoint_local": "fde4:8dba:82e1::9abc/64",
    "endpoint_remote": "fde4:8dba:82e1::def0/64",
    "delegated_prefix_1": "2a05:d014:1c0a:cd00::/64",
    "delegated_prefix_2": "2a12:5940:d891:cd00::/64",
    "delegated_prefix_3": "2a06:e881:7400:test::/64",
    "created_at": "2024-11-12T20:05:00Z"
    // Brak pól WireGuard
  }
}
```

---

## 📚 TypeScript Interfaces

```typescript
// Zaktualizowany interface
export interface Tunnel {
  id: string;
  user_id: string;
  type: 'sit' | 'gre' | 'wg';  // ⬅️ ZMIANA
  status: 'active' | 'suspended';
  server_ipv4: string;
  client_ipv4: string;
  endpoint_local: string;
  endpoint_remote: string;
  delegated_prefix_1: string;
  delegated_prefix_2: string;
  delegated_prefix_3: string;
  created_at: string;

  // ⬇️ NOWE pola (opcjonalne)
  server_private_key?: string;
  server_public_key?: string;
  client_private_key?: string;
  client_public_key?: string;
  listen_port?: number;
}

export interface TunnelCommands {
  server: string[];
  client: string[];
}

export interface TunnelResponse {
  tunnel: Tunnel;
  commands: TunnelCommands;
}

export interface CreateTunnelRequest {
  type: 'sit' | 'gre' | 'wg';  // ⬅️ ZMIANA
  user_id: string;
  client_ipv4: string;
}

// Type guard helper
export const isWireGuardTunnel = (tunnel: Tunnel): boolean => {
  return tunnel.type === 'wg';
};

// Type-safe access to WireGuard fields
export interface WireGuardTunnel extends Tunnel {
  type: 'wg';
  server_private_key: string;
  server_public_key: string;
  client_private_key: string;
  client_public_key: string;
  listen_port: number;
}
```

---

## ⚠️ Uwagi dotyczące bezpieczeństwa

### Klucze prywatne

1. **Nigdy nie loguj kluczy prywatnych** w console.log lub error logs
2. **Pokazuj klucze prywatne tylko za zgodą użytkownika** (kliknięcie "Show")
3. **Dodaj ostrzeżenie** przed pokazaniem klucza prywatnego
4. **Rozważ auto-hide** klucza po określonym czasie
5. **Nie przechowuj kluczy** w localStorage bez szyfrowania

### Przykład implementacji safe logging:

```typescript
const sanitizeTunnel = (tunnel: Tunnel): Partial<Tunnel> => {
  const { server_private_key, client_private_key, ...safe } = tunnel;
  return {
    ...safe,
    server_private_key: server_private_key ? '***REDACTED***' : undefined,
    client_private_key: client_private_key ? '***REDACTED***' : undefined,
  };
};

// Użycie:
console.log('Tunnel data:', sanitizeTunnel(tunnel));
```

---

## 🔗 Przydatne linki

- **WireGuard Protocol:** https://www.wireguard.com/protocol/
- **Curve25519:** https://cr.yp.to/ecdh.html
- **Migration Guide:** `/internal/db/migrations/README.md`
- **API Documentation:** (aktualizuj zgodnie z tym dokumentem)

---

## 📞 Wsparcie

Jeśli masz pytania dotyczące implementacji:
1. Sprawdź przykłady w tym dokumencie
2. Zobacz test cases w kodzie backendu
3. Skontaktuj się z zespołem backend

---

**Dokument wersja:** 1.0
**Data:** 2024-11-12
**Autor:** Backend Team
**Status:** ✅ Ready for Frontend Implementation
