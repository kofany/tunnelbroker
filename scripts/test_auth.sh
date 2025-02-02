#!/bin/bash

API_URL="http://localhost:9090"

echo "1. Rozpoczynanie procesu logowania..."
LOGIN_RESPONSE=$(curl -s -D - "${API_URL}/auth/login")
echo "$LOGIN_RESPONSE"

# Wyciągnij URL przekierowania
REDIRECT_URL=$(echo "$LOGIN_RESPONSE" | grep -i "location:" | cut -d' ' -f2)
echo "URL przekierowania: $REDIRECT_URL"

echo "2. Symulacja callback'u OAuth2..."
CALLBACK_RESPONSE=$(curl -s -D - "${API_URL}/auth/callback?code=test-code&state=test-state" \
    -H "Cookie: auth_state=test-state")
echo "$CALLBACK_RESPONSE"

# Wyciągnij token dostępu
ACCESS_TOKEN=$(echo "$CALLBACK_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
echo "Token dostępu: $ACCESS_TOKEN"

echo "3. Test chronionego endpointu z tokenem..."
curl -s -H "Authorization: Bearer $ACCESS_TOKEN" "${API_URL}/api/v1/system/status"
echo

echo "4. Test chronionego endpointu bez tokenu..."
curl -s "${API_URL}/api/v1/system/status"
echo

echo "5. Test chronionego endpointu z nieprawidłowym tokenem..."
curl -s -H "Authorization: Bearer invalid-token" "${API_URL}/api/v1/system/status"
echo 