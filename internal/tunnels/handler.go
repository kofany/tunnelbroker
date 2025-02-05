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

// CreateTunnelHandler handles POST /api/v1/tunnels request
func CreateTunnelHandler(c *gin.Context) {
	var req struct {
		Type       string `json:"type" binding:"required,oneof=sit gre"`
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

	if err := UpdateClientIPv4(tunnelID, req.ClientIPv4); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Dla uproszczenia zwracamy przykładowe komendy aktualizacji
	commands := TunnelCommands{
		Server: []string{
			"ip tunnel change " + tunnelID + " mode <type> remote " + req.ClientIPv4 + " ttl 255",
		},
		Client: []string{
			"ip tunnel change " + tunnelID + " mode <type> remote <server_ipv4> local " + req.ClientIPv4 + " ttl 255",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
	})
}

// DeleteTunnelHandler handles DELETE /api/v1/tunnels/:tunnel_id request
func DeleteTunnelHandler(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")

	// First remove the interface from the system
	command := exec.Command("ip", "tunnel", "del", tunnelID)
	if err := command.Run(); err != nil {
		applog.Logger.Printf("Error removing tunnel interface %s: %v", tunnelID, err)
		// Continue even if interface removal fails
	}

	// Then delete from database
	if err := DeleteTunnel(tunnelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
		commands := &TunnelCommands{}
		if strings.ToLower(t.Type) == "sit" {
			commands.Server = []string{
				fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", t.ID, t.ServerIPv4, t.ClientIPv4),
				fmt.Sprintf("ip link set %s up", t.ID),
				fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointLocal, t.ID),
				fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix1, t.ID),
				fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix2, t.ID),
			}
			commands.Client = []string{
				fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", t.ID, t.ClientIPv4, t.ServerIPv4),
				fmt.Sprintf("ip link set %s up", t.ID),
				fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointRemote, t.ID),
				fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix1, "/64"), t.ID),
				fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix2, "/64"), t.ID),
				fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(t.EndpointLocal, "/64"), t.ID),
			}
		} else if strings.ToLower(t.Type) == "gre" {
			commands.Server = []string{
				fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", t.ID, t.ServerIPv4, t.ClientIPv4),
				fmt.Sprintf("ip link set %s up", t.ID),
				fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointLocal, t.ID),
				fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix1, t.ID),
				fmt.Sprintf("ip -6 route add %s dev %s", t.DelegatedPrefix2, t.ID),
			}
			commands.Client = []string{
				fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", t.ID, t.ClientIPv4, t.ServerIPv4),
				fmt.Sprintf("ip link set %s up", t.ID),
				fmt.Sprintf("ip -6 addr add %s dev %s", t.EndpointRemote, t.ID),
				fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix1, "/64"), t.ID),
				fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(t.DelegatedPrefix2, "/64"), t.ID),
				fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(t.EndpointLocal, "/64"), t.ID),
			}
		}

		response = append(response, TunnelWithCommands{
			Tunnel:   t,
			Commands: *commands,
		})
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
	commands := &TunnelCommands{}
	if strings.ToLower(tunnel.Type) == "sit" {
		commands.Server = []string{
			fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", tunnel.ID, tunnel.ServerIPv4, tunnel.ClientIPv4),
			fmt.Sprintf("ip link set %s up", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", tunnel.EndpointLocal, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix1, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix2, tunnel.ID),
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", tunnel.ID, tunnel.ClientIPv4, tunnel.ServerIPv4),
			fmt.Sprintf("ip link set %s up", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", tunnel.EndpointRemote, tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(tunnel.DelegatedPrefix1, "/64"), tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(tunnel.DelegatedPrefix2, "/64"), tunnel.ID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(tunnel.EndpointLocal, "/64"), tunnel.ID),
		}
	} else if strings.ToLower(tunnel.Type) == "gre" {
		commands.Server = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", tunnel.ID, tunnel.ServerIPv4, tunnel.ClientIPv4),
			fmt.Sprintf("ip link set %s up", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", tunnel.EndpointLocal, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix1, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix2, tunnel.ID),
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", tunnel.ID, tunnel.ClientIPv4, tunnel.ServerIPv4),
			fmt.Sprintf("ip link set %s up", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", tunnel.EndpointRemote, tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(tunnel.DelegatedPrefix1, "/64"), tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(tunnel.DelegatedPrefix2, "/64"), tunnel.ID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(tunnel.EndpointLocal, "/64"), tunnel.ID),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tunnel":   tunnel,
		"commands": commands,
	})
}
