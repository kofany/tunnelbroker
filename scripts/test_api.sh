#!/bin/bash

API_URL="http://localhost:9090"
AUTH_TOKEN="test-token"  # W rzeczywistości należy uzyskać token przez OAuth2

# Funkcja do wykonywania żądań HTTP z autoryzacją
function call_api() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$data" ]; then
        curl -X "$method" \
             -H "Authorization: Bearer $AUTH_TOKEN" \
             -H "Content-Type: application/json" \
             "${API_URL}${endpoint}"
    else
        curl -X "$method" \
             -H "Authorization: Bearer $AUTH_TOKEN" \
             -H "Content-Type: application/json" \
             -d "$data" \
             "${API_URL}${endpoint}"
    fi
    echo
}

echo "1. Dodawanie głównego prefiksu IPv6..."
call_api POST "/api/v1/prefixes" '{
    "prefix": "2001:db8::/32",
    "description": "Test prefix pool"
}'

echo "2. Tworzenie tunelu SIT..."
TUNNEL_ID=$(call_api POST "/api/v1/tunnels" '{
    "type": "sit",
    "client_nickname": "test-client",
    "client_ipv4": "192.0.2.2",
    "server_ipv4": "192.0.2.1"
}' | jq -r '.id')

echo "Utworzony tunel ID: $TUNNEL_ID"

echo "3. Sprawdzanie statusu tunelu..."
call_api GET "/api/v1/tunnels/$TUNNEL_ID"

echo "4. Zawieszanie tunelu..."
call_api PUT "/api/v1/tunnels/$TUNNEL_ID/suspend"

echo "5. Aktywacja tunelu..."
call_api PUT "/api/v1/tunnels/$TUNNEL_ID/activate"

echo "6. Tworzenie tunelu GRE..."
GRE_TUNNEL_ID=$(call_api POST "/api/v1/tunnels" '{
    "type": "gre",
    "client_nickname": "test-gre-client",
    "client_ipv4": "192.0.2.3",
    "server_ipv4": "192.0.2.1"
}' | jq -r '.id')

echo "Utworzony tunel GRE ID: $GRE_TUNNEL_ID"

echo "7. Lista wszystkich tuneli..."
call_api GET "/api/v1/tunnels"

echo "8. Lista wszystkich prefiksów..."
call_api GET "/api/v1/prefixes"

echo "9. Pobieranie konfiguracji klienta..."
call_api GET "/api/v1/system/config/$TUNNEL_ID"

echo "10. Usuwanie tuneli..."
call_api DELETE "/api/v1/tunnels/$TUNNEL_ID"
call_api DELETE "/api/v1/tunnels/$GRE_TUNNEL_ID"

echo "11. Sprawdzanie statusu systemu..."
call_api GET "/api/v1/system/status" 