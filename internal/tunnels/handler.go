package tunnels

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateTunnelHandler obsługuje żądanie POST /api/v1/tunnels
func CreateTunnelHandler(c *gin.Context) {
	var req struct {
		Type       string `json:"type" binding:"required,oneof=sit gre"`
		UserID     string `json:"user_id" binding:"required,len=4"`
		ClientIPv4 string `json:"client_ipv4" binding:"required,ipv4"`
		ServerIPv4 string `json:"server_ipv4" binding:"required,ipv4"` // adres serwera, przekazywany w żądaniu
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tunnel, commands, err := CreateTunnelService(req.Type, req.UserID, req.ClientIPv4, req.ServerIPv4)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tunnel":   tunnel,
		"commands": commands,
	})
}

// UpdateClientIPHandler obsługuje żądanie PATCH /api/v1/tunnels/:tunnel_id/ip
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

// DeleteTunnelHandler obsługuje żądanie DELETE /api/v1/tunnels/:tunnel_id
func DeleteTunnelHandler(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")
	if err := DeleteTunnel(tunnelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
