package core

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/kofany/tunnelbroker/internal/state"
)

// SystemService zarządza operacjami systemowymi
type SystemService struct {
	stateManager *state.Manager
	configDir    string
}

// NewSystemService tworzy nową instancję serwisu systemowego
func NewSystemService(stateManager *state.Manager) *SystemService {
	return &SystemService{
		stateManager: stateManager,
		configDir:    "/etc/tunnelbroker/configs",
	}
}

// GenerateClientConfig generuje plik konfiguracyjny dla klienta
func (s *SystemService) GenerateClientConfig(tunnel state.Tunnel) error {
	// Upewnij się, że katalog konfiguracji istnieje
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(s.configDir, tunnel.ID+".yaml")
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	// Pobierz prefiksy przypisane do tunelu
	prefixes := s.stateManager.GetTunnelPrefixes(tunnel.ID)
	if len(prefixes) < 2 {
		return fmt.Errorf("tunnel must have at least 2 prefixes assigned")
	}

	// Przygotuj dane do szablonu
	data := struct {
		ID          string
		Type        string
		ClientIPv4  string
		ServerIPv4  string
		EndpointIP  string
		Prefixes    []string
	}{
		ID:          tunnel.ID,
		Type:        tunnel.Type,
		ClientIPv4:  tunnel.ClientIPv4,
		ServerIPv4:  tunnel.ServerIPv4,
		EndpointIP:  prefixes[0], // Pierwszy prefiks to endpoint
		Prefixes:    prefixes[1:], // Pozostałe prefiksy
	}

	// Szablon konfiguracji
	tmpl := template.Must(template.New("config").Parse(`tunnel:
  name: {{.ID}}
  type: {{.Type}}
  local_ip: {{.ClientIPv4}}
  remote_ip: {{.ServerIPv4}}
  ttl: 255

ipv6:
  endpoint: {{.EndpointIP}}/64
  prefixes:{{range .Prefixes}}
    - {{.}}/64{{end}}

routes:
  - ::/0 via {{.EndpointIP}}
`))

	return tmpl.Execute(file, data)
}

// CheckSystemHealth sprawdza stan systemu
func (s *SystemService) CheckSystemHealth() error {
	// Sprawdź dostęp do /etc
	if _, err := os.Stat("/etc"); err != nil {
		return fmt.Errorf("cannot access /etc: %v", err)
	}

	// Sprawdź dostęp do katalogu konfiguracji
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("cannot access config directory: %v", err)
	}

	// Sprawdź uprawnienia do wykonywania komend ip
	if _, err := os.Stat("/sbin/ip"); err != nil {
		return fmt.Errorf("cannot access ip command: %v", err)
	}

	return nil
}

// BackupConfigs tworzy kopię zapasową konfiguracji
func (s *SystemService) BackupConfigs() error {
	backupDir := filepath.Join(s.configDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Tutaj powinna być implementacja backupu
	return nil
}

// GetSystemMetrics zwraca metryki systemu
func (s *SystemService) GetSystemMetrics() (map[string]interface{}, error) {
	metrics := make(map[string]interface{})
	
	// Liczba aktywnych tuneli
	metrics["active_tunnels"] = len(s.stateManager.GetActiveTunnels())
	
	// Liczba głównych prefiksów
	metrics["main_prefixes"] = len(s.stateManager.GetMainPrefixes())

	return metrics, nil
} 