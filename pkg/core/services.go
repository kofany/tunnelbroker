package core

import (
	"github.com/kofany/tunnelbroker/internal/database"
	"github.com/kofany/tunnelbroker/internal/state"
)

// Services zawiera wszystkie główne serwisy aplikacji
type Services struct {
	stateManager  *state.Manager
	tunnelService *TunnelService
	prefixService *PrefixService
	systemService *SystemService
	db            *database.DB
}

// NewServices tworzy nową instancję serwisów
func NewServices(dbPath string) (*Services, error) {
	db, err := database.NewDB(dbPath)
	if err != nil {
		return nil, err
	}

	stateManager, err := state.NewManager()
	if err != nil {
		return nil, err
	}

	tunnelService := NewTunnelService(stateManager, db)
	prefixService := NewPrefixService(stateManager)
	systemService := NewSystemService(stateManager)

	return &Services{
		stateManager:  stateManager,
		tunnelService: tunnelService,
		prefixService: prefixService,
		systemService: systemService,
		db:            db,
	}, nil
}

// GetTunnelService zwraca serwis do zarządzania tunelami
func (s *Services) GetTunnelService() *TunnelService {
	return s.tunnelService
}

// GetPrefixService zwraca serwis do zarządzania prefiksami
func (s *Services) GetPrefixService() *PrefixService {
	return s.prefixService
}

// GetSystemService zwraca serwis systemowy
func (s *Services) GetSystemService() *SystemService {
	return s.systemService
}

// GetStateManager zwraca menedżera stanu
func (s *Services) GetStateManager() *state.Manager {
	return s.stateManager
}
