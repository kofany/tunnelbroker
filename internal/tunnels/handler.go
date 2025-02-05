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

// GetTunnelsHandler obsługuje żądanie GET /api/v1/tunnels
func GetTunnelsHandler(c *gin.Context) {
	// Sprawdź, czy żądanie zawiera user_id w query params
	userID := c.Query("user_id")

	var (
		tunnels []Tunnel
		err     error
	)

	if userID != "" {
		// Jeśli podano user_id, zwróć tylko tunele tego użytkownika
		tunnels, err = GetUserTunnels(userID)
	} else {
		// Jeśli nie podano user_id, zwróć wszystkie tunele (tylko dla admina)
		tunnels, err = GetAllTunnels()
	}

	if err != nil {
		applog.Logger.Printf("Błąd pobierania tuneli: %v", err)
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

// GetTunnelHandler obsługuje żądanie GET /api/v1/tunnels/:tunnel_id
func GetTunnelHandler(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")

	tunnel, err := GetTunnelByID(tunnelID)
	if err != nil {
		if strings.Contains(err.Error(), "nie znaleziono tunelu") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		applog.Logger.Printf("Błąd pobierania tunelu: %v", err)
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
