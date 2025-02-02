package state

import (
	"fmt"
	"sync"
	"time"
)

// MainPrefix reprezentuje główny prefiks IPv6
type MainPrefix struct {
	Prefix      string          `json:"prefix"`
	IsEndpoint  bool            `json:"is_endpoint"`
	Allocations map[string]bool `json:"allocations"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
}

// TunnelInfo reprezentuje informacje o tunelu
type TunnelInfo struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	UserID         int64     `json:"user_id"`
	ClientIPv4     string    `json:"client_ipv4"`
	ServerIPv4     string    `json:"server_ipv4"`
	Status         string    `json:"status"`
	EndpointPrefix string    `json:"endpoint_prefix"`
	Prefixes       []string  `json:"prefixes"`
	CreatedAt      time.Time `json:"created_at"`
	ModifiedAt     time.Time `json:"modified_at"`
}

// Manager zarządza stanem systemu
type Manager struct {
	mu            sync.RWMutex
	mainPrefixes  []MainPrefix
	activeTunnels []Tunnel
	lastOperation time.Time
}

// NewManager tworzy nową instancję menedżera stanu
func NewManager() (*Manager, error) {
	return &Manager{
		mainPrefixes:  make([]MainPrefix, 0),
		activeTunnels: make([]Tunnel, 0),
		lastOperation: time.Now(),
	}, nil
}

// GetMainPrefixes zwraca listę głównych prefiksów
func (m *Manager) GetMainPrefixes() []MainPrefix {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mainPrefixes
}

// GetActiveTunnels zwraca listę aktywnych tuneli
func (m *Manager) GetActiveTunnels() []Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeTunnels
}

// GetTunnel zwraca tunel o podanym ID
func (m *Manager) GetTunnel(id string) (Tunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.activeTunnels {
		if t.ID == id {
			return t, nil
		}
	}

	return Tunnel{}, fmt.Errorf("tunnel not found: %s", id)
}

// AddTunnel dodaje nowy tunel do systemu
func (m *Manager) AddTunnel(tunnel Tunnel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Sprawdź czy tunel o takim ID już istnieje
	for _, t := range m.activeTunnels {
		if t.ID == tunnel.ID {
			return fmt.Errorf("tunnel with ID %s already exists", tunnel.ID)
		}
	}

	m.activeTunnels = append(m.activeTunnels, tunnel)
	m.lastOperation = time.Now()
	return nil
}

// DeleteTunnel usuwa tunel z systemu
func (m *Manager) DeleteTunnel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, t := range m.activeTunnels {
		if t.ID == id {
			// Usuń tunel z listy
			m.activeTunnels = append(m.activeTunnels[:i], m.activeTunnels[i+1:]...)
			m.lastOperation = time.Now()
			return nil
		}
	}

	return fmt.Errorf("tunnel not found: %s", id)
}

// UpdateTunnelStatus aktualizuje status tunelu
func (m *Manager) UpdateTunnelStatus(id string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, t := range m.activeTunnels {
		if t.ID == id {
			m.activeTunnels[i].Status = status
			m.activeTunnels[i].ModifiedAt = time.Now()
			m.lastOperation = time.Now()
			return nil
		}
	}

	return fmt.Errorf("tunnel not found: %s", id)
}

// AddMainPrefix dodaje nowy główny prefiks
func (m *Manager) AddMainPrefix(prefix MainPrefix) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mainPrefixes = append(m.mainPrefixes, prefix)
	m.lastOperation = time.Now()
	return nil
}

// GetTunnelPrefixes zwraca listę prefiksów przypisanych do tunelu
func (m *Manager) GetTunnelPrefixes(tunnelID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var prefixes []string
	for _, t := range m.activeTunnels {
		if t.ID == tunnelID {
			// Dodaj prefiks endpointu
			if t.EndpointPrefix != "" {
				prefixes = append(prefixes, t.EndpointPrefix)
			}
			// Dodaj pozostałe prefiksy
			prefixes = append(prefixes, t.Prefix1, t.Prefix2)
			break
		}
	}
	return prefixes
}

func (m *Manager) GetTunnelInfo(tunnelID string) (TunnelInfo, error) {
	tunnel, err := m.GetTunnel(tunnelID)
	if err != nil {
		return TunnelInfo{}, err
	}

	info := TunnelInfo{
		ID:             tunnel.ID,
		Type:           tunnel.Type,
		UserID:         tunnel.UserID,
		ClientIPv4:     tunnel.ClientIPv4,
		ServerIPv4:     tunnel.ServerIPv4,
		Status:         tunnel.Status,
		EndpointPrefix: tunnel.EndpointPrefix,
		Prefixes:       []string{tunnel.Prefix1, tunnel.Prefix2},
		CreatedAt:      tunnel.CreatedAt,
		ModifiedAt:     tunnel.ModifiedAt,
	}

	return info, nil
}

func (m *Manager) GetTunnelByPrefix(prefix string) (Tunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.activeTunnels {
		if t.Prefix1 == prefix || t.Prefix2 == prefix {
			return t, nil
		}
	}

	return Tunnel{}, fmt.Errorf("no tunnel found for prefix: %s", prefix)
}
