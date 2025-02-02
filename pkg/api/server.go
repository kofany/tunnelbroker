package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kofany/tunnelbroker/internal/auth"
	"github.com/kofany/tunnelbroker/internal/state"
	"github.com/kofany/tunnelbroker/pkg/core"
)

// CORSMiddleware dodaje nagłówki CORS do odpowiedzi
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Config reprezentuje konfigurację serwera API
type Config struct {
	APIKeys struct {
		Frontend   APIKeyConfig
		Monitoring APIKeyConfig
	}
}

// Server reprezentuje serwer API
type Server struct {
	router     *mux.Router
	services   *core.Services
	httpServer *http.Server
	auth       *auth.Authenticator
	apiKeys    map[string]APIKeyConfig
}

// TunnelCommands reprezentuje komendy dla tunelu
type TunnelCommands struct {
	Server struct {
		ModuleLoad string   `json:"module_load"`
		Setup      []string `json:"setup"`
		Teardown   []string `json:"teardown"`
	} `json:"server"`
	Client struct {
		ModuleLoad      string   `json:"module_load"`
		Setup           []string `json:"setup"`
		DefaultRoute    string   `json:"default_route"`
		SelectiveRoutes []string `json:"selective_routes"`
	} `json:"client"`
	Info struct {
		TunnelID   string `json:"tunnel_id"`
		Type       string `json:"type"`
		ServerIPv4 string `json:"server_ipv4"`
		ClientIPv4 string `json:"client_ipv4"`
		ULAServer  string `json:"ula_server"`
		ULAClient  string `json:"ula_client"`
		Prefix1    string `json:"prefix1"`
		Prefix2    string `json:"prefix2"`
	} `json:"info"`
}

// NewServer tworzy nową instancję serwera API
func NewServer(services *core.Services, config *Config) *Server {
	s := &Server{
		router:   mux.NewRouter(),
		services: services,
		apiKeys: map[string]APIKeyConfig{
			config.APIKeys.Frontend.Key:   config.APIKeys.Frontend,
			config.APIKeys.Monitoring.Key: config.APIKeys.Monitoring,
		},
	}

	// Dodaj middleware CORS do wszystkich tras
	s.router.Use(CORSMiddleware)

	// Inicjalizacja middleware dla kluczy API
	apiKeyMiddleware, err := NewAPIKeyMiddleware(s.apiKeys)
	if err != nil {
		log.Fatalf("Failed to initialize API key middleware: %v", err)
	}

	// Dodaj middleware do wszystkich tras API
	apiRouter := s.router.PathPrefix("/api").Subrouter()
	apiRouter.Use(apiKeyMiddleware.Middleware)

	s.setupRoutes()
	return s
}

// UseAuth konfiguruje autoryzację dla serwera API
func (s *Server) UseAuth(authenticator *auth.Authenticator) {
	s.auth = authenticator

	// Dodaj trasy autoryzacji
	s.router.HandleFunc("/auth/login", authenticator.LoginHandler).Methods("GET")
	s.router.HandleFunc("/auth/callback", authenticator.CallbackHandler).Methods("GET")

	// Zabezpiecz wszystkie trasy API
	apiRouter := s.router.PathPrefix("/api").Subrouter()
	apiRouter.Use(authenticator.Middleware)

	// Przenieś wszystkie trasy API do zabezpieczonego routera
	s.setupProtectedRoutes(apiRouter)
}

// setupProtectedRoutes konfiguruje zabezpieczone trasy API
func (s *Server) setupProtectedRoutes(router *mux.Router) {
	// Trasy dla tuneli
	router.HandleFunc("/v1/tunnels", s.handleCreateTunnel).Methods("POST")
	router.HandleFunc("/v1/tunnels", s.handleListTunnels).Methods("GET")
	router.HandleFunc("/v1/tunnels/{id}", s.handleGetTunnel).Methods("GET")
	router.HandleFunc("/v1/tunnels/{id}", s.handleDeleteTunnel).Methods("DELETE")
	router.HandleFunc("/v1/tunnels/{id}/suspend", s.handleSuspendTunnel).Methods("PUT")
	router.HandleFunc("/v1/tunnels/{id}/activate", s.handleActivateTunnel).Methods("PUT")
	router.HandleFunc("/v1/tunnels/{id}/commands", s.handleGetTunnelCommands).Methods("GET")

	// Trasy dla prefiksów
	router.HandleFunc("/v1/prefixes", s.handleAddPrefix).Methods("POST")
	router.HandleFunc("/v1/prefixes", s.handleListPrefixes).Methods("GET")
	router.HandleFunc("/v1/prefixes/endpoint", s.handleSetEndpointPrefix).Methods("PUT")

	// Trasy systemowe
	router.HandleFunc("/v1/system/status", s.handleSystemStatus).Methods("GET")
	router.HandleFunc("/v1/system/config/{id}", s.handleGetConfig).Methods("GET")
}

// setupRoutes konfiguruje podstawowe trasy
func (s *Server) setupRoutes() {
	// Trasa główna
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("TunnelBroker API"))
	}).Methods("GET")

	// Trasa healthcheck
	s.router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	// Konfiguracja tras API z autoryzacją po kluczu API
	apiRouter := s.router.PathPrefix("/api").Subrouter()
	s.setupProtectedRoutes(apiRouter)
}

// Start uruchamia serwer API
func (s *Server) Start(ctx context.Context) error {
	s.httpServer = &http.Server{
		Addr:    ":9090",
		Handler: s.router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(shutdownCtx)
	}()

	return s.httpServer.ListenAndServe()
}

// handleCreateTunnel obsługuje tworzenie nowego tunelu
func (s *Server) handleCreateTunnel(w http.ResponseWriter, r *http.Request) {
	var tunnel state.Tunnel
	if err := json.NewDecoder(r.Body).Decode(&tunnel); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Walidacja typu tunelu
	if tunnel.Type != "sit" && tunnel.Type != "gre" {
		http.Error(w, "invalid tunnel type, must be 'sit' or 'gre'", http.StatusBadRequest)
		return
	}

	// Ustaw domyślne wartości
	tunnel.Status = "active"
	tunnel.CreatedAt = time.Now()
	tunnel.ModifiedAt = time.Now()

	// Utwórz tunel i pobierz konfigurację dla klienta
	config, err := s.services.GetTunnelService().CreateTunnel(tunnel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(config)
}

// handleListTunnels obsługuje listowanie tuneli
func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	tunnels := s.services.GetTunnelService().GetActiveTunnels()
	json.NewEncoder(w).Encode(tunnels)
}

// handleGetTunnel obsługuje pobieranie informacji o tunelu
func (s *Server) handleGetTunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]

	tunnel, err := s.services.GetTunnelService().GetTunnel(tunnelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(tunnel)
}

// handleDeleteTunnel obsługuje usuwanie tunelu
func (s *Server) handleDeleteTunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]

	if err := s.services.GetTunnelService().DeleteTunnel(tunnelID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSuspendTunnel obsługuje zawieszanie tunelu
func (s *Server) handleSuspendTunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]

	if err := s.services.GetTunnelService().SuspendTunnel(tunnelID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleActivateTunnel obsługuje aktywację tunelu
func (s *Server) handleActivateTunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]

	if err := s.services.GetTunnelService().ActivateTunnel(tunnelID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleAddPrefix obsługuje dodawanie prefiksu
func (s *Server) handleAddPrefix(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prefix      string `json:"prefix"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.services.GetPrefixService().AddMainPrefix(req.Prefix, req.Description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// handleListPrefixes obsługuje listowanie prefiksów
func (s *Server) handleListPrefixes(w http.ResponseWriter, r *http.Request) {
	prefixes := s.services.GetPrefixService().GetMainPrefixes()
	json.NewEncoder(w).Encode(prefixes)
}

// handleSetEndpointPrefix obsługuje ustawianie prefiksu końcowego
func (s *Server) handleSetEndpointPrefix(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prefix string `json:"prefix"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.services.GetPrefixService().SetEndpointPrefix(req.Prefix); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleSystemStatus obsługuje sprawdzanie statusu systemu
func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tunnels := s.services.GetTunnelService().GetActiveTunnels()
	prefixes := s.services.GetPrefixService().GetMainPrefixes()

	response := map[string]interface{}{
		"total_tunnels":  len(tunnels),
		"active_tunnels": len(tunnels),
		"total_prefixes": len(prefixes),
		"status":         "operational",
		"version":        "1.0.0",
		"server_ipv4":    "192.67.35.38",
	}

	json.NewEncoder(w).Encode(response)
}

// handleGetConfig obsługuje pobieranie konfiguracji klienta
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]

	// Pobierz tunel
	tunnel, err := s.services.GetTunnelService().GetTunnel(tunnelID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Tunnel not found: %s", tunnelID), http.StatusNotFound)
		return
	}

	// Wygeneruj konfigurację
	if err := s.services.GetSystemService().GenerateClientConfig(tunnel); err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate config: %v", err), http.StatusInternalServerError)
		return
	}

	// Odczytaj wygenerowany plik
	configPath := filepath.Join("/etc/tunnelbroker/configs", tunnelID+".yaml")
	config, err := os.ReadFile(configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.yaml", tunnelID))
	w.Write(config)
}

// handleGetTunnelCommands obsługuje pobieranie komend dla tunelu
func (s *Server) handleGetTunnelCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]

	tunnel, err := s.services.GetTunnelService().GetTunnel(tunnelID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Tunnel not found: %s", tunnelID), http.StatusNotFound)
		return
	}

	commands := TunnelCommands{}

	// Wypełnij informacje o tunelu
	commands.Info.TunnelID = tunnel.ID
	commands.Info.Type = tunnel.Type
	commands.Info.ServerIPv4 = tunnel.ServerIPv4
	commands.Info.ClientIPv4 = tunnel.ClientIPv4
	commands.Info.ULAServer = tunnel.EndpointPrefix
	commands.Info.ULAClient = strings.Replace(tunnel.EndpointPrefix, "::1", "::2", 1)
	commands.Info.Prefix1 = tunnel.Prefix1
	commands.Info.Prefix2 = tunnel.Prefix2

	// Ustaw komendy w zależności od typu tunelu
	if tunnel.Type == "sit" {
		commands.Server.ModuleLoad = "modprobe sit"
		commands.Client.ModuleLoad = "modprobe sit"

		// Komendy dla serwera
		commands.Server.Setup = []string{
			fmt.Sprintf("ip tunnel add %s mode sit remote %s local %s ttl 64",
				tunnel.ID, tunnel.ClientIPv4, tunnel.ServerIPv4),
			fmt.Sprintf("ip link set %s up mtu 1480", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s nodad", tunnel.EndpointPrefix, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s metric 1", tunnel.Prefix1, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s metric 1", tunnel.Prefix2, tunnel.ID),
		}

		// Komendy dla klienta
		commands.Client.Setup = []string{
			fmt.Sprintf("ip tunnel add %s mode sit remote %s local %s ttl 64",
				tunnel.ID, tunnel.ServerIPv4, tunnel.ClientIPv4),
			fmt.Sprintf("ip link set %s up mtu 1480", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s nodad",
				strings.Replace(tunnel.EndpointPrefix, "::1", "::2", 1), tunnel.ID),
		}
	} else if tunnel.Type == "gre" {
		commands.Server.ModuleLoad = "modprobe ip_gre"
		commands.Client.ModuleLoad = "modprobe ip_gre"

		// Komendy dla serwera
		commands.Server.Setup = []string{
			fmt.Sprintf("ip tunnel add %s mode gre remote %s local %s ttl 64",
				tunnel.ID, tunnel.ClientIPv4, tunnel.ServerIPv4),
			fmt.Sprintf("ip link set %s up mtu 1480", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s nodad", tunnel.EndpointPrefix, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s metric 1", tunnel.Prefix1, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s metric 1", tunnel.Prefix2, tunnel.ID),
		}

		// Komendy dla klienta
		commands.Client.Setup = []string{
			fmt.Sprintf("ip tunnel add %s mode gre remote %s local %s ttl 64",
				tunnel.ID, tunnel.ServerIPv4, tunnel.ClientIPv4),
			fmt.Sprintf("ip link set %s up mtu 1480", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s nodad",
				strings.Replace(tunnel.EndpointPrefix, "::1", "::2", 1), tunnel.ID),
		}
	}

	// Wspólne dla obu typów
	commands.Server.Teardown = []string{
		fmt.Sprintf("ip tunnel del %s", tunnel.ID),
	}

	// Trasy dla klienta
	commands.Client.DefaultRoute = fmt.Sprintf("ip -6 route add ::/0 via %s dev %s metric 1",
		strings.Replace(tunnel.EndpointPrefix, "/64", "", 1), tunnel.ID)
	commands.Client.SelectiveRoutes = []string{
		fmt.Sprintf("ip -6 route add %s dev %s metric 1", tunnel.Prefix1, tunnel.ID),
		fmt.Sprintf("ip -6 route add %s dev %s metric 1", tunnel.Prefix2, tunnel.ID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commands)
}
