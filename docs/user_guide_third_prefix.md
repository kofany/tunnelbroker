# TunnelBroker User Guide: Third IPv6 Prefix

## Overview

TunnelBroker now provides three IPv6 /64 prefixes for each tunnel, enhancing your IPv6 connectivity options. This guide explains how to use and configure the new third prefix.

## Prefix Allocation

Each tunnel now receives:

1. **First Prefix**: From primary /44 range (e.g., 2a05:dfc1:3c0F:abcd::/64)
2. **Second Prefix**: From secondary /44 range (e.g., 2a12:bec0:2c0F:abcd::/64)
3. **New Third Prefix**: From dedicated /48 range (e.g., 2a06:1234:5600:abcd::/64)

The third prefix follows a simpler format that directly embeds your user ID, making it more predictable and easier to remember.

## Tunnel Configuration

### Server-side Configuration

The server automatically configures routing for all three prefixes. No action is required on your part.

### Client-side Configuration

When creating a new tunnel, you'll receive configuration commands that include the third prefix. Here's an example for a SIT tunnel:

```bash
# Set up the tunnel interface
ip tunnel add tun-abcd-1 mode sit local 141.11.62.211 remote 192.67.35.38 ttl 255
ip link set tun-abcd-1 up

# Configure the tunnel endpoint
ip -6 addr add fde4:5a50:1114:beef::2/64 dev tun-abcd-1

# Configure all three delegated prefixes
ip -6 addr add 2a05:dfc1:3c0F:abcd:1/64 dev tun-abcd-1
ip -6 addr add 2a12:bec0:2c0F:abcd:1/64 dev tun-abcd-1
ip -6 addr add 2a06:1234:5600:abcd:1/64 dev tun-abcd-1

# Set up default route
ip -6 route add ::/0 via fde4:5a50:1114:beef::1 dev tun-abcd-1
```

## Using Your Third Prefix

The third prefix can be used just like the first two prefixes. Here are some common use cases:

### Subnet Allocation

You can divide your /64 prefixes for different purposes:

- First prefix: Primary network
- Second prefix: IoT devices
- Third prefix: Guest network or specialized services

### Network Segmentation

Use the third prefix for network segmentation to enhance security:

```bash
# Create a VLAN interface
ip link add link eth0 name eth0.100 type vlan id 100
ip link set eth0.100 up

# Assign the third prefix to the VLAN
ip -6 addr add 2a06:1234:5600:abcd:1/64 dev eth0.100
```

### Redundancy and Failover

Configure failover between prefixes for critical services:

```bash
# Primary service using first prefix
ip -6 addr add 2a05:dfc1:3c0F:abcd:1:1:1:1/64 dev eth0

# Backup service using third prefix
ip -6 addr add 2a06:1234:5600:abcd:1:1:1:1/64 dev eth0
```

## Existing Tunnels

If you have existing tunnels, they will continue to work with two prefixes. To get the third prefix, you can:

1. Recreate your tunnel
2. Contact support to have the third prefix added to your existing tunnel

## Troubleshooting

If you encounter issues with your third prefix:

1. Verify that your tunnel configuration includes all three prefixes
2. Check that routing is properly configured for the third prefix
3. Ensure that your firewall allows traffic on the third prefix
4. Test connectivity using ping6 to a known IPv6 address

For further assistance, please contact support with your tunnel ID and the specific issue you're experiencing.
