package tunnels

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kofany/tunnelbroker/internal/config"
	applog "github.com/kofany/tunnelbroker/internal/logger"
)

// executeCommands wykonuje listę komend systemowych
func executeCommands(commands []string) error {
	for _, cmd := range commands {
		command := exec.Command("sh", "-c", cmd)
		if err := command.Run(); err != nil {
			applog.Logger.Printf("Error executing command %s: %v", cmd, err)
			return err
		}
	}
	return nil
}

// generateTunnelCommands generates commands for a given tunnel based on its type
func generateTunnelCommands(t *Tunnel) *TunnelCommands {
	commands := &TunnelCommands{}

	switch strings.ToLower(t.Type) {
	case "sit":
		commands.Server = []string{
			fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", t.ID, t.ServerIPv4, t.ClientIPv4),
			fmt.Sprintf("ip link set %s up", t.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointLocal, t.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix1, t.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix2, t.ID),
		}
		if t.DelegatedPrefix3 != "" {
			commands.Server = append(commands.Server, fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix3, t.ID))
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", t.ID, t.ClientIPv4, t.ServerIPv4),
			fmt.Sprintf("ip link set %s up", t.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointRemote, t.ID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix1, "/64"), t.ID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix2, "/64"), t.ID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(t.EndpointLocal, "/64"), t.ID),
		}
		if t.DelegatedPrefix3 != "" {
			commands.Client = append(commands.Client, fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix3, "/64"), t.ID))
		}

	case "gre":
		commands.Server = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", t.ID, t.ServerIPv4, t.ClientIPv4),
			fmt.Sprintf("ip link set %s up", t.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointLocal, t.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix1, t.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix2, t.ID),
		}
		if t.DelegatedPrefix3 != "" {
			commands.Server = append(commands.Server, fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix3, t.ID))
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", t.ID, t.ClientIPv4, t.ServerIPv4),
			fmt.Sprintf("ip link set %s up", t.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointRemote, t.ID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix1, "/64"), t.ID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix2, "/64"), t.ID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(t.EndpointLocal, "/64"), t.ID),
		}
		if t.DelegatedPrefix3 != "" {
			commands.Client = append(commands.Client, fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix3, "/64"), t.ID))
		}

	case "wg":
		// WireGuard uses single wg0 interface with multiple peers
		wgInterface := config.GlobalConfig.WireGuard.Interface
		wgPort := config.GlobalConfig.WireGuard.ListenPort
		serverPubKey := config.GlobalConfig.WireGuard.PublicKey

		// Build allowed-ips list including all prefixes
		allowedIPs := fmt.Sprintf("%s,%s", t.EndpointRemote, t.DelegatedPrefix1)
		if t.DelegatedPrefix2 != "" {
			allowedIPs += "," + t.DelegatedPrefix2
		}
		if t.DelegatedPrefix3 != "" {
			allowedIPs += "," + t.DelegatedPrefix3
		}

		// Server-side: add peer to wg0 and routes
		commands.Server = []string{
			fmt.Sprintf("wg set %s peer %s allowed-ips %s", wgInterface, t.ClientPublicKey, allowedIPs),
			fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix1, wgInterface),
			fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix2, wgInterface),
		}
		if t.DelegatedPrefix3 != "" {
			commands.Server = append(commands.Server, fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix3, wgInterface))
		}

		// Client-side commands (for user to run)
		clientInterface := t.ID // Client uses tunnel ID as interface name
		commands.Client = []string{
			fmt.Sprintf("ip link add dev %s type wireguard", clientInterface),
			fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointRemote, clientInterface),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix1, "/64"), clientInterface),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix2, "/64"), clientInterface),
		}
		if t.DelegatedPrefix3 != "" {
			commands.Client = append(commands.Client, fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix3, "/64"), clientInterface))
		}
		// WireGuard config command - client stores private key in file
		commands.Client = append(commands.Client,
			fmt.Sprintf("echo '%s' > /etc/wireguard/%s_private.key && chmod 600 /etc/wireguard/%s_private.key", t.ClientPrivateKey, clientInterface, clientInterface),
			fmt.Sprintf("wg set %s private-key /etc/wireguard/%s_private.key peer %s endpoint %s:%d allowed-ips ::/0",
				clientInterface, clientInterface, serverPubKey, t.ServerIPv4, wgPort),
			fmt.Sprintf("ip link set %s up", clientInterface),
			fmt.Sprintf("ip -6 route add ::/0 dev %s", clientInterface),
		)
	}

	return commands
}

// CreateTunnelHandler handles POST /api/v1/tunnels request
func CreateTunnelHandler(c *gin.Context) {
	var req struct {
		Type       string `json:"type" binding:"required,oneof=sit gre wg"`
		UserID     string `json:"user_id" binding:"required,len=4"`
		ClientIPv4 string `json:"client_ipv4" binding:"required,ipv4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		applog.Logger.Printf("Request validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get server_ipv4 from configuration
	serverIPv4 := config.GlobalConfig.Server.IPv4
	if serverIPv4 == "" {
		applog.Logger.Printf("Error: missing server_ipv4 configuration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing server_ipv4 configuration"})
		return
	}

	tunnel, commands, err := CreateTunnelService(req.Type, req.UserID, req.ClientIPv4, serverIPv4)
	if err != nil {
		applog.Logger.Printf("Error creating tunnel: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Execute server-side commands
	if err := executeCommands(commands.Server); err != nil {
		applog.Logger.Printf("Error executing server commands: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tunnel configuration error"})
		return
	}

	// Apply security rules
	securityCmd := exec.Command("/etc/tunnelbroker/scripts/tunnel_security.sh")
	if err := securityCmd.Run(); err != nil {
		applog.Logger.Printf("Error applying security rules: %v", err)
		// Continue even if security script fails
	}

	c.JSON(http.StatusOK, gin.H{
		"tunnel":   tunnel,
		"commands": commands,
	})
}

// UpdateClientIPHandler handles PATCH /api/v1/tunnels/:tunnel_id/ip request
func UpdateClientIPHandler(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")
	var req struct {
		ClientIPv4 string `json:"client_ipv4" binding:"required,ipv4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Aktualizacja IP w bazie danych
	if err := UpdateClientIPv4(tunnelID, req.ClientIPv4); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Pobranie zaktualizowanego tunelu z bazy danych
	tunnel, err := GetTunnelByID(tunnelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generowanie komend zależnie od typu tunelu
	var commands TunnelCommands

	tunnelType := strings.ToLower(tunnel.Type)
	if tunnelType == "wg" {
		// WireGuard: no server-side command needed for IP update
		// WireGuard automatically discovers client endpoint after handshake
		// We only update the database entry
		commands.Server = []string{} // No-op on server side

		// Client commands for reference (new endpoint configuration)
		wgPort := config.GlobalConfig.WireGuard.ListenPort
		commands.Client = []string{
			fmt.Sprintf("# WireGuard automatically handles endpoint changes after handshake"),
			fmt.Sprintf("# If needed, reconfigure: wg set %s peer <server_pubkey> endpoint %s:%d",
				tunnel.ID, tunnel.ServerIPv4, wgPort),
		}
	} else {
		// SIT/GRE tunnels
		commands.Server = []string{
			fmt.Sprintf("ip tunnel change %s mode %s remote %s ttl 255",
				tunnel.ID, tunnelType, req.ClientIPv4),
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel change %s mode %s remote %s local %s ttl 255",
				tunnel.ID, tunnelType, tunnel.ServerIPv4, req.ClientIPv4),
		}
	}

	// Wykonanie komend systemowych (tylko dla nie-WG)
	if len(commands.Server) > 0 && tunnelType != "wg" {
		if err := executeCommands(commands.Server); err != nil {
			applog.Logger.Printf("Error updating tunnel interface: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Tunnel update error"})
			return
		}
	}

	// Zastosuj również reguły bezpieczeństwa po aktualizacji
	securityCmd := exec.Command("/etc/tunnelbroker/scripts/tunnel_security.sh")
	if err := securityCmd.Run(); err != nil {
		applog.Logger.Printf("Warning: Error applying security rules after IP update: %v", err)
		// Continue even if security script fails
	}

	c.JSON(http.StatusOK, gin.H{
		"tunnel":   tunnel,
		"commands": commands,
	})
}

// DeleteTunnelHandler handles DELETE /api/v1/tunnels/:tunnel_id request
func DeleteTunnelHandler(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")

	// Get tunnel info before deletion to retrieve user_id
	tunnel, err := GetTunnelByID(tunnelID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tunnel not found"})
		} else {
			applog.Logger.Printf("Error retrieving tunnel info for deletion: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Remove tunnel from system
	tunnelType := strings.ToLower(tunnel.Type)
	if tunnelType == "wg" {
		// WireGuard: remove peer from wg0 and routes
		wgInterface := config.GlobalConfig.WireGuard.Interface

		// Remove peer from wg0
		peerCmd := exec.Command("wg", "set", wgInterface, "peer", tunnel.ClientPublicKey, "remove")
		if err := peerCmd.Run(); err != nil {
			applog.Logger.Printf("Error removing WireGuard peer %s: %v", tunnel.ClientPublicKey, err)
		}

		// Remove routes
		for _, prefix := range []string{tunnel.DelegatedPrefix1, tunnel.DelegatedPrefix2, tunnel.DelegatedPrefix3} {
			if prefix != "" {
				routeCmd := exec.Command("ip", "-6", "route", "del", prefix, "dev", wgInterface)
				if err := routeCmd.Run(); err != nil {
					applog.Logger.Printf("Error removing route %s: %v", prefix, err)
				}
			}
		}
	} else {
		// SIT and GRE use 'ip tunnel del'
		command := exec.Command("ip", "tunnel", "del", tunnelID)
		if err := command.Run(); err != nil {
			applog.Logger.Printf("Error removing tunnel interface %s: %v", tunnelID, err)
			// Continue even if interface removal fails
		}
	}

	// Then delete from database
	if err := DeleteTunnel(tunnelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update user's active tunnels counter
	if err := DecrementActiveUserTunnels(tunnel.UserID); err != nil {
		applog.Logger.Printf("Error updating user tunnels counter: %v", err)
		// Continue even if counter update fails
	}

	// Apply security script to refresh the rules and clean up any remaining rules
	securityCmd := exec.Command("/etc/tunnelbroker/scripts/tunnel_security.sh")
	if err := securityCmd.Run(); err != nil {
		applog.Logger.Printf("Error applying security rules after tunnel deletion: %v", err)
		// Continue even if security script fails
	}

	c.Status(http.StatusNoContent)
}

// GetTunnelsHandler handles GET /api/v1/tunnels request
func GetTunnelsHandler(c *gin.Context) {
	// Check if request contains user_id in query params
	userID := c.Query("user_id")

	var (
		tunnels []Tunnel
		err     error
	)

	if userID != "" {
		// If user_id is provided, return only user's tunnels
		tunnels, err = GetUserTunnels(userID)
	} else {
		// If no user_id is provided, return all tunnels (admin only)
		tunnels, err = GetAllTunnels()
	}

	if err != nil {
		applog.Logger.Printf("Error retrieving tunnels: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Dla każdego tunelu wygeneruj komendy
	type TunnelWithCommands struct {
		Tunnel   Tunnel         `json:"tunnel"`
		Commands TunnelCommands `json:"commands"`
	}

	var response []TunnelWithCommands
	for _, t := range tunnels {
		tCopy := t // Create a copy to pass pointer
		commands := generateTunnelCommands(&tCopy)

		response = append(response, TunnelWithCommands{
			Tunnel:   t,
			Commands: *commands,
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetUserTunnelsHandler handles GET /api/v1/tunnels/user/:user_id request
func GetUserTunnelsHandler(c *gin.Context) {
	userID := c.Param("user_id")

	// Validate user_id format (should be 4 characters)
	if len(userID) != 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format. Must be 4 characters."})
		return
	}

	// Get tunnels for the specified user
	tunnels, err := GetUserTunnels(userID)
	if err != nil {
		applog.Logger.Printf("Error retrieving user tunnels: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get user information
	user, err := GetUserByID(userID)
	if err != nil {
		applog.Logger.Printf("Error retrieving user info: %v", err)
		// Continue even if user info retrieval fails, just log the error
		user = &User{
			ID:             userID,
			CreatedTunnels: 0,
			ActiveTunnels:  0,
		}
	}

	// For each tunnel, generate commands
	type TunnelWithCommands struct {
		Tunnel   Tunnel         `json:"tunnel"`
		Commands TunnelCommands `json:"commands"`
	}

	var tunnelsWithCommands []TunnelWithCommands
	for _, t := range tunnels {
		tCopy := t // Create a copy to pass pointer
		commands := generateTunnelCommands(&tCopy)

		tunnelsWithCommands = append(tunnelsWithCommands, TunnelWithCommands{
			Tunnel:   t,
			Commands: *commands,
		})
	}

	// Create response with tunnels and user info
	response := gin.H{
		"tunnels": tunnelsWithCommands,
		"user_info": gin.H{
			"created_tunnels": user.CreatedTunnels,
			"active_tunnels":  user.ActiveTunnels,
		},
	}

	// If no tunnels found, return empty array for tunnels
	if len(tunnels) == 0 {
		response["tunnels"] = []any{}
	}

	c.JSON(http.StatusOK, response)
}

// GetTunnelHandler handles GET /api/v1/tunnels/:tunnel_id request
func GetTunnelHandler(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")

	tunnel, err := GetTunnelByID(tunnelID)
	if err != nil {
		if strings.Contains(err.Error(), "tunnel not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		applog.Logger.Printf("Error retrieving tunnel: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generowanie komend dla tunelu
	commands := generateTunnelCommands(tunnel)

	c.JSON(http.StatusOK, gin.H{
		"tunnel":   tunnel,
		"commands": commands,
	})
}
