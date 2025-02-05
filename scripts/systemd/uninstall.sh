#!/bin/bash

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root"
  exit 1
fi

# Stop and disable service
echo "Stopping and disabling service..."
systemctl stop tunnelbroker
systemctl disable tunnelbroker

# Remove systemd service
echo "Removing systemd service..."
rm -f /etc/systemd/system/tunnelbroker.service
systemctl daemon-reload

# Remove binary
echo "Removing binary..."
rm -f /usr/local/bin/tunnelbroker

# Remove configuration
echo "Removing configuration..."
rm -rf /etc/tunnelbroker

echo "Uninstallation complete!" 