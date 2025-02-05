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

# Build and install binary
echo "Building tunnelbroker..."
go build -o /usr/local/bin/tunnelbroker cmd/tunnelbroker/main.go

# Install systemd service
echo "Installing systemd service..."
cp scripts/systemd/tunnelbroker.service /etc/systemd/system/

# Reload systemd
systemctl daemon-reload

# Enable and start service
echo "Enabling and starting service..."
systemctl enable tunnelbroker
systemctl start tunnelbroker

echo "Installation complete!"
echo "Please check service status with: systemctl status tunnelbroker" 