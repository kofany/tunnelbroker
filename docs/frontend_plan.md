# Plan implementacji frontendu TunnelBroker

## 1. Technologie
- Next.js 14 (React + TypeScript)
- Tailwind CSS
- Supabase (tylko auth)
- Vercel (hosting)

## 2. Autentykacja (Supabase)

### Struktura danych
```sql
-- Tylko dane użytkowników
auth.users (tabela Supabase):
  • id (UUID)
  • email (string)
  • full_name (string)
  • avatar_url (string)
  • role (enum: 'user', 'admin')
```

### Metody logowania
- Google OAuth
- Facebook OAuth
- GitHub OAuth

## 3. Struktura aplikacji

```
src/
├── components/           # Komponenty współdzielone
├── pages/               # Strony aplikacji
├── hooks/               # Custom hooks
├── lib/                 # Utilities
└── api/                 # Integracja z backendem
```

## 4. Integracja z backendem

### Konfiguracja API
```typescript
// api/tunnelbroker.ts
const api = {
  baseUrl: process.env.NEXT_PUBLIC_API_URL,
  headers: {
    'X-API-Key': process.env.API_KEY,
    'Content-Type': 'application/json'
  }
}
```

### Endpointy i ich użycie

#### 1. Tworzenie tunelu
```typescript
// API Call
POST /api/v1/tunnels
Body: {
  type: "sit" | "gre",
  user_id: number,
  client_ipv4: string,
  server_ipv4: string
}

// Użycie w komponencie
const CreateTunnel = () => {
  const createTunnel = async (data) => {
    const response = await fetch('/api/v1/tunnels', {
      method: 'POST',
      headers: api.headers,
      body: JSON.stringify({
        ...data,
        user_id: supabase.auth.user().id
      })
    });
    return response.json();
  };
};
```

#### 2. Lista tuneli użytkownika
```typescript
// API Call
GET /api/v1/tunnels

// Użycie w komponencie
const TunnelList = () => {
  const [tunnels, setTunnels] = useState([]);
  
  useEffect(() => {
    const fetchTunnels = async () => {
      const response = await fetch('/api/v1/tunnels', {
        headers: api.headers
      });
      setTunnels(await response.json());
    };
    fetchTunnels();
  }, []);
};
```

#### 3. Szczegóły tunelu
```typescript
// API Call
GET /api/v1/tunnels/{id}
GET /api/v1/tunnels/{id}/commands

// Użycie w komponencie
const TunnelDetails = ({ id }) => {
  const [tunnel, setTunnel] = useState(null);
  const [commands, setCommands] = useState(null);
  
  useEffect(() => {
    const fetchData = async () => {
      const [tunnelRes, commandsRes] = await Promise.all([
        fetch(`/api/v1/tunnels/${id}`),
        fetch(`/api/v1/tunnels/${id}/commands`)
      ]);
      setTunnel(await tunnelRes.json());
      setCommands(await commandsRes.json());
    };
    fetchData();
  }, [id]);
};
```

#### 4. Zarządzanie tunelem
```typescript
// API Calls
DELETE /api/v1/tunnels/{id}
PUT /api/v1/tunnels/{id}/suspend
PUT /api/v1/tunnels/{id}/activate

// Użycie w komponencie
const TunnelActions = ({ id }) => {
  const deleteTunnel = () => fetch(`/api/v1/tunnels/${id}`, {
    method: 'DELETE',
    headers: api.headers
  });
  
  const suspendTunnel = () => fetch(`/api/v1/tunnels/${id}/suspend`, {
    method: 'PUT',
    headers: api.headers
  });
  
  const activateTunnel = () => fetch(`/api/v1/tunnels/${id}/activate`, {
    method: 'PUT',
    headers: api.headers
  });
};
```

## 5. Widoki

### Dashboard (/pages/dashboard/index.tsx)
- Wyświetla listę tuneli użytkownika
- Pokazuje limit tuneli (max 2)
- Przycisk tworzenia nowego tunelu (jeśli < 2)
- Status każdego tunelu

### Tworzenie tunelu (/pages/dashboard/tunnels/create.tsx)
- Formularz z wyborem typu (SIT/GRE)
- Automatyczna detekcja IP klienta
- Walidacja danych
- Przekierowanie po utworzeniu

### Szczegóły tunelu (/pages/dashboard/tunnels/[id].tsx)
- Informacje o tunelu
- Komendy konfiguracyjne
- Przyciski akcji (suspend/activate/delete)
- Kopiowanie komend do schowka

## 6. Panel admina (/pages/admin/*)

### Lista użytkowników
```typescript
// Używa danych z Supabase
const UserList = () => {
  const { data: users } = await supabase
    .from('users')
    .select('*');
};
```

### Lista wszystkich tuneli
```typescript
// API Call
GET /api/v1/tunnels (z headerem admin)

// Użycie w komponencie
const AdminTunnelList = () => {
  // Podobnie jak TunnelList, ale z dodatkowymi informacjami
};
```

## 7. Wdrożenie

### Vercel
- Automatyczne deploye z GitHuba
- Zmienne środowiskowe:
  ```
  NEXT_PUBLIC_SUPABASE_URL=xxx
  NEXT_PUBLIC_SUPABASE_ANON_KEY=xxx
  NEXT_PUBLIC_API_URL=xxx
  API_KEY=xxx
  ```

### CI/CD
- Testy przed deployem
- Automatyczne buildy
- Preview deployments

## 8. Bezpieczeństwo

### Middleware
```typescript
// middleware.ts
export function middleware(req: NextRequest) {
  // Sprawdź auth
  const session = await getSession(req);
  if (!session) return redirectToLogin(req);
  
  // Sprawdź limity dla zwykłych użytkowników
  if (req.url.includes('/api/v1/tunnels') && 
      session.user.role !== 'admin') {
    const tunnels = await getTunnelCount(session.user.id);
    if (tunnels >= 2) return new Response('Limit exceeded', { status: 403 });
  }
}
```

### Rate Limiting
- Wykorzystujemy rate limiting z backendu
- Dodatkowy limit na froncie dla API calls 