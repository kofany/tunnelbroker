#!/bin/bash

# Sprawdź czy skrypt jest uruchomiony jako root
if [ "$EUID" -ne 0 ]; then
    echo "Ten skrypt musi być uruchomiony jako root"
    exit 1
fi

# Stwórz katalogi
mkdir -p /etc/tunnelbroker
mkdir -p /var/lib/tunnelbroker
mkdir -p /var/log/tunnelbroker
mkdir -p /usr/local/bin

# Skopiuj pliki
cp tunnelbroker /usr/local/bin/
cp scripts/tunnelbroker.service /etc/systemd/system/
cp config.yaml /etc/tunnelbroker/

# Ustaw uprawnienia
chmod 755 /usr/local/bin/tunnelbroker
chmod 644 /etc/systemd/system/tunnelbroker.service
chmod 644 /etc/tunnelbroker/config.yaml
chown -R root:root /etc/tunnelbroker
chown -R root:root /var/lib/tunnelbroker
chown -R root:root /var/log/tunnelbroker

# Załaduj i uruchom usługę
systemctl daemon-reload
systemctl enable tunnelbroker
systemctl start tunnelbroker

echo "TunnelBroker został zainstalowany i uruchomiony"
echo "Sprawdź status: systemctl status tunnelbroker" 