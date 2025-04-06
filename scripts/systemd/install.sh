#!/bin/bash

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root"
  exit 1
fi

# Create configuration directory
mkdir -p /etc/tunnelbroker

# Copy configuration files
if [ -f .env ]; then
    cp .env /etc/tunnelbroker/
else
    echo "Warning: .env file not found"
fi

if [ -f cmd/config/config.yaml ]; then
    cp cmd/config/config.yaml /etc/tunnelbroker/
else
    echo "Warning: config.yaml file not found"
fi

# Install required packages
apt-get update
apt-get install -y iptables-persistent iproute2

# Build and install binary
echo "Building tunnelbroker and tunnelrecovery..."
go build -o /usr/local/bin/tunnelbroker cmd/tunnelbroker/main.go
go build -o /usr/local/bin/tunnelrecovery cmd/tunnelrecovery/main.go

# Install systemd service
echo "Installing systemd service..."
cp scripts/systemd/tunnelbroker.service /etc/systemd/system/

# Copy and configure security script
mkdir -p /etc/tunnelbroker/scripts
cp scripts/security/tunnel_security.sh /etc/tunnelbroker/scripts/
chmod +x /etc/tunnelbroker/scripts/tunnel_security.sh

# Create directory for iptables rules
mkdir -p /etc/iptables

# Reload systemd
systemctl daemon-reload

# Enable and start service
echo "Enabling and starting service..."
systemctl enable tunnelbroker
systemctl start tunnelbroker

# Configure security for existing tunnels
echo "Configuring security rules..."
/etc/tunnelbroker/scripts/tunnel_security.sh

echo "Installation complete!"
echo "Please check service status with: systemctl status tunnelbroker" 