package tunnels

import (
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/kofany/tunnelbroker/internal/config"
	applog "github.com/kofany/tunnelbroker/internal/logger"
)

// executeCommands wykonuje listę komend systemowych
func executeCommands(commands []string) error {
	for _, cmd := range commands {
		command := exec.Command("sh", "-c", cmd)
		if err := command.Run(); err != nil {
			applog.Logger.Printf("Błąd wykonania komendy %s: %v", cmd, err)
			return err
		}
	}
	return nil
}

// CreateTunnelHandler obsługuje żądanie POST /api/v1/tunnels
func CreateTunnelHandler(c *gin.Context) {
	var req struct {
		Type       string `json:"type" binding:"required,oneof=sit gre"`
		UserID     string `json:"user_id" binding:"required,len=4"`
		ClientIPv4 string `json:"client_ipv4" binding:"required,ipv4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		applog.Logger.Printf("Błąd walidacji żądania: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Pobierz server_ipv4 z konfiguracji
	serverIPv4 := config.GlobalConfig.Server.IPv4
	if serverIPv4 == "" {
		applog.Logger.Printf("Błąd: brak konfiguracji server_ipv4")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brak konfiguracji server_ipv4"})
		return
	}

	tunnel, commands, err := CreateTunnelService(req.Type, req.UserID, req.ClientIPv4, serverIPv4)
	if err != nil {
		applog.Logger.Printf("Błąd tworzenia tunelu: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Wykonaj komendy po stronie serwera
	if err := executeCommands(commands.Server); err != nil {
		applog.Logger.Printf("Błąd wykonania komend serwera: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Błąd konfiguracji tunelu"})
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

	// Najpierw usuń interfejs z systemu
	command := exec.Command("ip", "tunnel", "del", tunnelID)
	if err := command.Run(); err != nil {
		applog.Logger.Printf("Błąd usuwania interfejsu tunelu %s: %v", tunnelID, err)
		// Kontynuujemy nawet jeśli nie udało się usunąć interfejsu
	}

	// Następnie usuń z bazy danych
	if err := DeleteTunnel(tunnelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
