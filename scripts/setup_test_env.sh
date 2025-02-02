#!/bin/bash

# Konfiguracja środowiska testowego
export OIDC_ISSUER_URL="https://accounts.google.com/.well-known/openid-configuration"  # Pełny URL konfiguracji OIDC
export OIDC_CLIENT_ID="test-client-id"
export OIDC_CLIENT_SECRET="test-client-secret"
export OIDC_REDIRECT_URL="http://localhost:9090/auth/callback"

# Tworzenie katalogu dla bazy danych
mkdir -p /etc/tunnelbroker/configs
chmod 755 /etc/tunnelbroker/configs

# Upewnienie się, że moduły jądra są załadowane
modprobe sit
modprobe gre
modprobe ipv6

# Włączenie przekazywania IPv6
sysctl -w net.ipv6.conf.all.forwarding=1

# Konfiguracja testowego adresu IPv4 dla serwera
ip addr add 192.0.2.1/24 dev lo

# Tworzenie testowego prefiksu IPv6
ip -6 addr add 2001:db8::/32 dev lo 