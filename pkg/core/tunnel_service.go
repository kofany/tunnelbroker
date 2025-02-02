package core

import (
	"fmt"
	"os/exec"

	"github.com/kofany/tunnelbroker/internal/interfaces"
	"github.com/kofany/tunnelbroker/internal/state"
)

// TunnelType określa typ tunelu
type TunnelType string

const (
	TunnelTypeSIT TunnelType = "sit"
	TunnelTypeGRE TunnelType = "gre"
)

// TunnelConfig zawiera konfigurację dla klienta
type TunnelConfig struct {
	TunnelID          string   `json:"tunnel_id"`
	ServerIPv4        string   `json:"server_ipv4"`
	ClientIPv4        string   `json:"client_ipv4"`
	EndpointLocal     string   `json:"endpoint_local"`
	EndpointRemote    string   `json:"endpoint_remote"`
	DelegatedPrefixes []string `json:"delegated_prefixes"`
	Routes            []Route  `json:"routes"`
}

type Route struct {
	Prefix string `json:"prefix"`
	Via    string `json:"via"`
}

// TunnelService zarządza operacjami na tunelach IPv6
type TunnelService struct {
	stateManager *state.Manager
	prefixPool   *state.PrefixPool
	db           interfaces.Database
}

// NewTunnelService tworzy nową instancję serwisu tuneli
func NewTunnelService(stateManager *state.Manager, db interfaces.Database) *TunnelService {
	return &TunnelService{
		stateManager: stateManager,
		prefixPool:   state.NewPrefixPool(),
		db:           db,
	}
}

// GetActiveTunnels zwraca listę aktywnych tuneli
func (s *TunnelService) GetActiveTunnels() []state.Tunnel {
	return s.stateManager.GetActiveTunnels()
}

// CreateTunnel tworzy nowy tunel IPv6 i zwraca konfigurację dla klienta
func (s *TunnelService) CreateTunnel(tunnel state.Tunnel) (*TunnelConfig, error) {
	// Generuj ID tunelu na podstawie typu i ID użytkownika
	if tunnel.ID == "" {
		if tunnel.Type == string(TunnelTypeGRE) {
			tunnel.ID = fmt.Sprintf("tun-gre%d", tunnel.UserID)
		} else {
			tunnel.ID = fmt.Sprintf("tun%d", tunnel.UserID)
		}
	}

	// Alokuj prefiksy IPv6
	endpointLocal, endpointRemote, prefix1, prefix2, err := s.prefixPool.AllocateForTunnel(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate prefixes: %v", err)
	}

	// Ustaw prefiksy w tunelu
	tunnel.EndpointPrefix = endpointLocal
	tunnel.Prefix1 = prefix1
	tunnel.Prefix2 = prefix2

	// Sprawdź typ tunelu
	tunnelType := TunnelTypeSIT
	if tunnel.Type == string(TunnelTypeGRE) {
		tunnelType = TunnelTypeGRE
	}

	// Przygotuj podstawowe parametry tunelu
	args := []string{
		"tunnel", "add", tunnel.ID,
		"mode", string(tunnelType),
		"remote", tunnel.ClientIPv4,
		"local", tunnel.ServerIPv4,
		"ttl", "255",
	}

	// Dodaj specyficzne parametry dla typu tunelu
	if tunnel.Type == string(TunnelTypeGRE) {
		args = append(args, "hoplimit", "64")
	}

	// Tworzenie tunelu w systemie
	cmd := exec.Command("ip", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Wycofaj alokację prefiksów w przypadku błędu
		s.prefixPool.ReleasePrefixes(prefix1, prefix2)
		return nil, fmt.Errorf("failed to create tunnel: %v (output: %s)", err, string(output))
	}

	// Aktywacja interfejsu
	cmd = exec.Command("ip", "link", "set", tunnel.ID, "up", "mtu", "1480")
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Wycofaj alokację prefiksów i usuń tunel w przypadku błędu
		s.prefixPool.ReleasePrefixes(prefix1, prefix2)
		exec.Command("ip", "tunnel", "del", tunnel.ID).Run()
		return nil, fmt.Errorf("failed to activate interface: %v (output: %s)", err, string(output))
	}

	// Konfiguracja adresu IPv6 dla endpointu
	cmd = exec.Command("ip", "-6", "addr", "add", endpointLocal,
		"dev", tunnel.ID, "nodad")
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Wycofaj wszystkie zmiany w przypadku błędu
		s.prefixPool.ReleasePrefixes(prefix1, prefix2)
		exec.Command("ip", "tunnel", "del", tunnel.ID).Run()
		return nil, fmt.Errorf("failed to configure IPv6 address: %v (output: %s)", err, string(output))
	}

	// Konfiguracja tras dla przydzielonych prefiksów
	for _, prefix := range []string{prefix1, prefix2} {
		cmd = exec.Command("ip", "-6", "route", "add", prefix,
			"dev", tunnel.ID, "metric", "1")
		output, err = cmd.CombinedOutput()
		if err != nil {
			// Wycofaj wszystkie zmiany w przypadku błędu
			s.prefixPool.ReleasePrefixes(prefix1, prefix2)
			exec.Command("ip", "tunnel", "del", tunnel.ID).Run()
			return nil, fmt.Errorf("failed to configure route for prefix %s: %v (output: %s)", prefix, err, string(output))
		}
	}

	// Zapisz tunel w stanie
	if err := s.stateManager.AddTunnel(tunnel); err != nil {
		// Wycofaj wszystkie zmiany w przypadku błędu
		s.prefixPool.ReleasePrefixes(prefix1, prefix2)
		exec.Command("ip", "tunnel", "del", tunnel.ID).Run()
		return nil, fmt.Errorf("failed to save tunnel state: %v", err)
	}

	// Przygotuj konfigurację dla klienta
	config := &TunnelConfig{
		TunnelID:          tunnel.ID,
		ServerIPv4:        tunnel.ServerIPv4,
		ClientIPv4:        tunnel.ClientIPv4,
		EndpointLocal:     endpointRemote, // Dla klienta lokalny to nasz zdalny
		EndpointRemote:    endpointLocal,  // Dla klienta zdalny to nasz lokalny
		DelegatedPrefixes: []string{prefix1, prefix2},
		Routes: []Route{
			{Prefix: prefix1, Via: endpointLocal},
			{Prefix: prefix2, Via: endpointLocal},
		},
	}

	return config, nil
}

// DeleteTunnel usuwa tunel IPv6
func (s *TunnelService) DeleteTunnel(tunnelID string) error {
	tunnel, err := s.stateManager.GetTunnel(tunnelID)
	if err != nil {
		return err
	}

	// Usuń trasy IPv6
	for _, prefix := range []string{tunnel.Prefix1, tunnel.Prefix2} {
		cmd := exec.Command("ip", "-6", "route", "del", prefix, "dev", tunnelID)
		cmd.Run() // Ignorujemy błędy, bo trasa może już nie istnieć
	}

	// Usuń adres IPv6
	if tunnel.EndpointPrefix != "" {
		cmd := exec.Command("ip", "-6", "addr", "del", tunnel.EndpointPrefix, "dev", tunnelID)
		cmd.Run() // Ignorujemy błędy, bo adres może już nie istnieć
	}

	// Dezaktywuj interfejs
	cmd := exec.Command("ip", "link", "set", tunnelID, "down")
	if err := cmd.Run(); err != nil {
		// Logujemy błąd, ale kontynuujemy
		fmt.Printf("Warning: failed to set interface down: %v\n", err)
	}

	// Usuń tunel
	cmd = exec.Command("ip", "tunnel", "del", tunnelID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete tunnel: %v (output: %s)", err, string(output))
	}

	// Zwolnij prefiksy
	s.prefixPool.ReleasePrefixes(tunnel.Prefix1, tunnel.Prefix2)

	return s.stateManager.DeleteTunnel(tunnelID)
}

// SuspendTunnel zawiesza działanie tunelu
func (s *TunnelService) SuspendTunnel(tunnelID string) error {
	tunnel, err := s.stateManager.GetTunnel(tunnelID)
	if err != nil {
		return err
	}

	// Usuń trasy IPv6
	for _, prefix := range []string{tunnel.Prefix1, tunnel.Prefix2} {
		cmd := exec.Command("ip", "-6", "route", "del", prefix, "dev", tunnelID)
		cmd.Run() // Ignorujemy błędy, bo trasa może już nie istnieć
	}

	// Dezaktywuj interfejs
	cmd := exec.Command("ip", "link", "set", tunnelID, "down")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to suspend tunnel: %v (output: %s)", err, string(output))
	}

	return s.stateManager.UpdateTunnelStatus(tunnelID, "suspended")
}

// ActivateTunnel wznawia działanie tunelu
func (s *TunnelService) ActivateTunnel(tunnelID string) error {
	tunnel, err := s.stateManager.GetTunnel(tunnelID)
	if err != nil {
		return err
	}

	// Aktywuj interfejs
	cmd := exec.Command("ip", "link", "set", tunnelID, "up")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to activate interface: %v (output: %s)", err, string(output))
	}

	// Przywróć trasy IPv6
	for _, prefix := range []string{tunnel.Prefix1, tunnel.Prefix2} {
		cmd = exec.Command("ip", "-6", "route", "add", prefix,
			"dev", tunnelID, "metric", "1")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to restore route for prefix %s: %v (output: %s)", prefix, err, string(output))
		}
	}

	return s.stateManager.UpdateTunnelStatus(tunnelID, "active")
}

// GetTunnelStatus zwraca status tunelu
func (s *TunnelService) GetTunnelStatus(tunnelID string) (string, error) {
	output, err := exec.Command("ip", "link", "show", tunnelID).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get tunnel status: %v", err)
	}

	// Sprawdź czy interfejs jest aktywny
	if len(output) > 0 {
		return "active", nil
	}
	return "inactive", nil
}

// UpdateTunnelConfig aktualizuje konfigurację tunelu
func (s *TunnelService) UpdateTunnelConfig(tunnel state.Tunnel) error {
	// Najpierw usuń stary tunel
	if err := s.DeleteTunnel(tunnel.ID); err != nil {
		return fmt.Errorf("failed to delete old tunnel: %v", err)
	}

	// Utwórz nowy tunel z zaktualizowaną konfiguracją
	_, err := s.CreateTunnel(tunnel)
	return err
}

// GetTunnel zwraca informacje o tunelu
func (s *TunnelService) GetTunnel(tunnelID string) (state.Tunnel, error) {
	return s.stateManager.GetTunnel(tunnelID)
}
