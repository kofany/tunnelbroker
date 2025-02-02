package core

import (
	"fmt"
	"net"
	"time"

	"github.com/kofany/tunnelbroker/internal/state"
)

// PrefixService zarządza prefiksami IPv6
type PrefixService struct {
	stateManager *state.Manager
}

// NewPrefixService tworzy nową instancję serwisu prefiksów
func NewPrefixService(stateManager *state.Manager) *PrefixService {
	return &PrefixService{
		stateManager: stateManager,
	}
}

// GetMainPrefixes zwraca listę głównych prefiksów
func (s *PrefixService) GetMainPrefixes() []state.MainPrefix {
	return s.stateManager.GetMainPrefixes()
}

// AddMainPrefix dodaje nowy główny prefiks
func (s *PrefixService) AddMainPrefix(prefix string, description string) error {
	// Walidacja prefiksu IPv6
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("invalid IPv6 prefix: %v", err)
	}

	mainPrefix := state.MainPrefix{
		Prefix:      ipNet.String(),
		IsEndpoint:  false,
		Allocations: make(map[string]bool),
		Description: description,
		CreatedAt:   time.Now(),
	}

	return s.stateManager.AddMainPrefix(mainPrefix)
}

// AllocatePrefix alokuje nowy prefiks /64 z puli głównej
func (s *PrefixService) AllocatePrefix(mainPrefix string) (string, error) {
	prefixes := s.stateManager.GetMainPrefixes()
	
	for _, p := range prefixes {
		if p.Prefix == mainPrefix {
			// Tutaj powinna być logika alokacji podprefiksu /64
			// Na potrzeby przykładu zwracamy błąd
			return "", fmt.Errorf("prefix allocation not implemented")
		}
	}

	return "", fmt.Errorf("main prefix not found")
}

// SetEndpointPrefix ustawia prefiks dla punktów końcowych
func (s *PrefixService) SetEndpointPrefix(prefix string) error {
	// Walidacja prefiksu IPv6
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("invalid IPv6 prefix: %v", err)
	}

	endpointPrefix := state.MainPrefix{
		Prefix:      ipNet.String(),
		IsEndpoint:  true,
		Allocations: make(map[string]bool),
		Description: "Endpoint prefix pool",
		CreatedAt:   time.Now(),
	}

	return s.stateManager.AddMainPrefix(endpointPrefix)
}

// ValidatePrefix sprawdza poprawność prefiksu IPv6
func (s *PrefixService) ValidatePrefix(prefix string) error {
	_, _, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("invalid IPv6 prefix: %v", err)
	}
	return nil
} 
