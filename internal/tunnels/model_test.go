package tunnels

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test for Tunnel struct
func TestTunnelStruct(t *testing.T) {
	// Create a sample tunnel
	tunnel := Tunnel{
		ID:               "tun-abcd-1",
		UserID:           "abcd",
		Type:             "sit",
		Status:           "active",
		ServerIPv4:       "192.67.35.38",
		ClientIPv4:       "141.11.62.211",
		EndpointLocal:    "fde4:5a50:1114:beef::1/64",
		EndpointRemote:   "fde4:5a50:1114:beef::2/64",
		DelegatedPrefix1: "2a05:dfc1:3c0F:abcd::/64",
		DelegatedPrefix2: "2a12:bec0:2c0F:abcd::/64",
		DelegatedPrefix3: "2a06:1234:5600:abcd::/64",
		CreatedAt:        time.Now(),
	}

	// Verify tunnel properties
	assert.Equal(t, "tun-abcd-1", tunnel.ID)
	assert.Equal(t, "abcd", tunnel.UserID)
	assert.Equal(t, "sit", tunnel.Type)
	assert.Equal(t, "active", tunnel.Status)
	assert.Equal(t, "192.67.35.38", tunnel.ServerIPv4)
	assert.Equal(t, "141.11.62.211", tunnel.ClientIPv4)
	assert.Equal(t, "fde4:5a50:1114:beef::1/64", tunnel.EndpointLocal)
	assert.Equal(t, "fde4:5a50:1114:beef::2/64", tunnel.EndpointRemote)
	assert.Equal(t, "2a05:dfc1:3c0F:abcd::/64", tunnel.DelegatedPrefix1)
	assert.Equal(t, "2a12:bec0:2c0F:abcd::/64", tunnel.DelegatedPrefix2)
	assert.Equal(t, "2a06:1234:5600:abcd::/64", tunnel.DelegatedPrefix3)
	assert.NotZero(t, tunnel.CreatedAt)
}

// Test for TunnelCommands struct
func TestTunnelCommandsStruct(t *testing.T) {
	// Create sample commands
	commands := TunnelCommands{
		Server: []string{
			"ip tunnel add tun-abcd-1 mode sit local 192.67.35.38 remote 141.11.62.211 ttl 255",
			"ip link set tun-abcd-1 up",
			"ip -6 addr add fde4:5a50:1114:beef::1/64 dev tun-abcd-1",
			"ip -6 route add 2a05:dfc1:3c0F:abcd::/64 dev tun-abcd-1",
			"ip -6 route add 2a12:bec0:2c0F:abcd::/64 dev tun-abcd-1",
			"ip -6 route add 2a06:1234:5600:abcd::/64 dev tun-abcd-1",
		},
		Client: []string{
			"ip tunnel add tun-abcd-1 mode sit local 141.11.62.211 remote 192.67.35.38 ttl 255",
			"ip link set tun-abcd-1 up",
			"ip -6 addr add fde4:5a50:1114:beef::2/64 dev tun-abcd-1",
			"ip -6 addr add 2a05:dfc1:3c0F:abcd:1/64 dev tun-abcd-1",
			"ip -6 addr add 2a12:bec0:2c0F:abcd:1/64 dev tun-abcd-1",
			"ip -6 addr add 2a06:1234:5600:abcd:1/64 dev tun-abcd-1",
			"ip -6 route add ::/0 via fde4:5a50:1114:beef::1 dev tun-abcd-1",
		},
	}

	// Verify server commands
	assert.Len(t, commands.Server, 6)
	assert.Contains(t, commands.Server[0], "ip tunnel add")
	assert.Contains(t, commands.Server[1], "ip link set")
	assert.Contains(t, commands.Server[2], "ip -6 addr add")
	assert.Contains(t, commands.Server[3], "ip -6 route add")
	assert.Contains(t, commands.Server[4], "ip -6 route add")
	assert.Contains(t, commands.Server[5], "ip -6 route add")

	// Verify client commands
	assert.Len(t, commands.Client, 7)
	assert.Contains(t, commands.Client[0], "ip tunnel add")
	assert.Contains(t, commands.Client[1], "ip link set")
	assert.Contains(t, commands.Client[2], "ip -6 addr add")
	assert.Contains(t, commands.Client[3], "ip -6 addr add")
	assert.Contains(t, commands.Client[4], "ip -6 addr add")
	assert.Contains(t, commands.Client[5], "ip -6 addr add")
	assert.Contains(t, commands.Client[6], "ip -6 route add ::/0")

	// Verify third prefix is included in commands
	assert.Contains(t, commands.Server[5], "2a06:1234:5600:abcd::/64")
	assert.Contains(t, commands.Client[5], "2a06:1234:5600:abcd:1/64")
}
