# Instrukcje testowania podstawowej funkcjonalności tuneli IPv6

## Cel
Stworzenie i przeprowadzenie kompleksowych testów dla głównych operacji na tunelach: tworzenie, usuwanie i zarządzanie.

## Zakres testów

### 1. Testy tworzenia tunelu (CreateTunnelService)

Przypadki testowe:
- Utworzenie pierwszego tunelu dla użytkownika
- Utworzenie drugiego tunelu dla użytkownika
- Próba utworzenia trzeciego tunelu (powinien być błąd)
- Tworzenie tuneli typu SIT i GRE
- Sprawdzenie poprawności generowanych komend
- Walidacja wszystkich przydzielonych prefixów
- Sprawdzenie unikalności ID tuneli

### 2. Testy usuwania tunelu (DeleteTunnel)

Przypadki testowe:
- Usunięcie istniejącego tunelu
- Próba usunięcia nieistniejącego tunelu
- Sprawdzenie czy system prawidłowo czyści zasoby
- Weryfikacja czy licznik created_tunnels nie jest zmniejszany
- Sprawdzenie czy prefixy są prawidłowo zwalniane

### 3. Przykładowa struktura testu

```go
func TestTunnelLifecycle(t *testing.T) {
    tests := []struct {
        name        string
        tunnelType  string
        userID      string
        clientIPv4  string
        serverIPv4  string
        operation   string // "create" lub "delete"
        wantErr     bool
        errorMsg    string
    }{
        {
            name: "create first tunnel",
            // ...
        },
        {
            name: "create second tunnel",
            // ...
        },
        {
            name: "attempt third tunnel",
            // ...
        },
        {
            name: "delete existing tunnel",
            // ...
        },
        {
            name: "delete non-existent tunnel",
            // ...
        },
    }
}
```

### 4. Elementy do sprawdzenia

Dla każdego utworzonego tunelu weryfikuj:
- Poprawność ID tunelu (format: tun-{user_id}-{number})
- Prawidłowość przydzielonych prefixów
- Poprawność komend systemowych
- Stan w bazie danych
- Licznik tuneli użytkownika

Dla każdego usuniętego tunelu weryfikuj:
- Prawidłowe usunięcie z bazy
- Generowanie właściwych komend systemowych
- Zachowanie licznika created_tunnels
- Brak osieroconych zasobów

### 5. Komendy do wykonania testów

```bash
# Uruchomienie wszystkich testów
go test -v ./internal/tunnels/...

# Uruchomienie konkretnego testu
go test -v -run TestTunnelLifecycle ./internal/tunnels/...

# Sprawdzenie pokrycia
go test -coverprofile=coverage.out ./internal/tunnels/...
go tool cover -html=coverage.out
```

### 6. Oczekiwane wyniki

Każdy test powinien:
- Weryfikować stan przed operacją
- Wykonywać testowaną operację
- Sprawdzać stan po operacji
- Czyścić środowisko testowe

### 7. Mock-i i stubs

Przygotuj mock-i dla:
- Wykonywania komend systemowych
- Operacji bazodanowych
- Generowania unikalnych identyfikatorów

## Kryteria akceptacji
- Wszystkie testy przechodzą
- Pokrycie kodu > 80%
- Testy sprawdzają zarówno pozytywne jak i negatywne przypadki
- Kod testowy jest czytelny i dobrze udokumentowany

## Uwagi
- Używaj testowych konfiguracji prefixów
- Izoluj testy od rzeczywistego systemu
- Implementuj pomocnicze funkcje testowe dla powtarzalnych operacji