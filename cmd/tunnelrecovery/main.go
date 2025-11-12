package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/kofany/tunnelbroker/internal/config"
)

// Tunnel represents a tunnel configuration
type Tunnel struct {
	ID               string `json:"id"`
	UserID           string `json:"user_id"`
	Type             string `json:"type"`
	Status           string `json:"status"`
	ServerIPv4       string `json:"server_ipv4"`
	ClientIPv4       string `json:"client_ipv4"`
	EndpointLocal    string `json:"endpoint_local"`
	EndpointRemote   string `json:"endpoint_remote"`
	DelegatedPrefix1 string `json:"delegated_prefix_1"`
	DelegatedPrefix2 string `json:"delegated_prefix_2"`
	DelegatedPrefix3 string `json:"delegated_prefix_3"`
	// WireGuard specific fields
	ServerPrivateKey string `json:"server_private_key,omitempty"`
	ServerPublicKey  string `json:"server_public_key,omitempty"`
	ClientPrivateKey string `json:"client_private_key,omitempty"`
	ClientPublicKey  string `json:"client_public_key,omitempty"`
	ListenPort       int    `json:"listen_port,omitempty"`
}

// TunnelResponse represents the API response structure
type TunnelResponse struct {
	Tunnel   Tunnel   `json:"tunnel"`
	Commands Commands `json:"commands"`
}

// Commands represents the commands for tunnel configuration
type Commands struct {
	Server []string `json:"server"`
	Client []string `json:"client"`
}

// Logger for syslog
var logger *log.Logger

func init() {
	var err error
	logger, err = syslog.NewLogger(syslog.LOG_NOTICE|syslog.LOG_DAEMON, log.LstdFlags)
	if err != nil {
		log.Fatalf("Failed to initialize syslog: %v", err)
	}
}

func main() {
	logger.Println("TunnelRecovery: Starting tunnel recovery process")

	// Load configuration
	if err := config.LoadConfig("/etc/tunnelbroker/config.yaml"); err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Wait for TunnelBroker service to start
	logger.Println("TunnelRecovery: Waiting for TunnelBroker service to start...")
	time.Sleep(5 * time.Second)

	// Get list of tunnels from API with retries
	var tunnels []Tunnel
	var err error
	for i := 0; i < 3; i++ { // Try 3 times
		tunnels, err = getTunnels()
		if err == nil {
			break
		}
		logger.Printf("Attempt %d: Failed to get tunnels from API: %v. Retrying in 2 seconds...", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		logger.Fatalf("Failed to get tunnels from API after multiple attempts: %v", err)
	}

	logger.Printf("TunnelRecovery: Found %d tunnels in database", len(tunnels))

	// Get existing tunnels in system
	existingTunnels, err := getExistingTunnels()
	if err != nil {
		logger.Fatalf("Failed to get existing tunnels: %v", err)
	}

	logger.Printf("TunnelRecovery: Found %d tunnels in system", len(existingTunnels))

	// Recreate missing tunnels
	recreatedCount := 0
	for _, tunnel := range tunnels {
		if tunnel.Status != "active" {
			logger.Printf("TunnelRecovery: Skipping inactive tunnel %s", tunnel.ID)
			continue
		}

		if !contains(existingTunnels, tunnel.ID) {
			logger.Printf("TunnelRecovery: Recreating missing tunnel %s", tunnel.ID)
			if err := recreateTunnel(tunnel); err != nil {
				logger.Printf("TunnelRecovery: Failed to recreate tunnel %s: %v", tunnel.ID, err)
			} else {
				recreatedCount++
			}
		}
	}

	logger.Printf("TunnelRecovery: Recovery completed. Recreated %d tunnels", recreatedCount)
}

// getTunnels retrieves tunnels from the API
func getTunnels() ([]Tunnel, error) {
	apiURL := fmt.Sprintf("http://%s/api/v1/tunnels", config.GlobalConfig.API.Listen)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-API-Key", config.GlobalConfig.API.Key)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
	}

	// For debugging
	respBody, _ := io.ReadAll(resp.Body)
	logger.Printf("API Response: %s", string(respBody))

	// Create a new reader with the response body
	reader := bytes.NewReader(respBody)

	// Parse the response as an array of TunnelResponse
	var tunnelResponses []TunnelResponse
	if err := json.NewDecoder(reader).Decode(&tunnelResponses); err != nil {
		return nil, fmt.Errorf("error decoding response: %w, body: %s", err, string(respBody))
	}

	// Extract tunnels from the response
	tunnels := make([]Tunnel, len(tunnelResponses))
	for i, resp := range tunnelResponses {
		tunnels[i] = resp.Tunnel
		logger.Printf("Tunnel found: %s (status: %s)", resp.Tunnel.ID, resp.Tunnel.Status)
		logger.Printf("Server commands: %v", resp.Commands.Server)
	}

	return tunnels, nil
}

// getExistingTunnels gets a list of existing tunnels in the system
func getExistingTunnels() ([]string, error) {
	var tunnels []string

	// Get SIT and GRE tunnels
	cmd := exec.Command("ip", "tunnel", "show")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		logger.Printf("Warning: error running 'ip tunnel show': %v", err)
	} else {
		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				tunnelName := strings.TrimSpace(parts[0])
				if tunnelName != "sit0" { // Skip the default sit0 interface
					tunnels = append(tunnels, tunnelName)
				}
			}
		}
	}

	// Get WireGuard tunnels
	wgCmd := exec.Command("ip", "link", "show", "type", "wireguard")
	var wgOut bytes.Buffer
	wgCmd.Stdout = &wgOut
	if err := wgCmd.Run(); err != nil {
		// WireGuard might not be available or no interfaces exist
		logger.Printf("Info: No WireGuard interfaces found or WireGuard not available")
	} else {
		lines := strings.Split(wgOut.String(), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			// WireGuard interfaces appear as: "N: interfacename: <POINTOPOINT,NOARP,UP,LOWER_UP> ..."
			if strings.Contains(line, ":") && !strings.HasPrefix(strings.TrimSpace(line), "link/") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					// Extract interface name (between first number and second colon)
					interfaceName := strings.TrimSpace(parts[1])
					if interfaceName != "" && strings.HasPrefix(interfaceName, "tun-") {
						tunnels = append(tunnels, interfaceName)
					}
				}
			}
		}
	}

	return tunnels, nil
}

// recreateTunnel recreates a tunnel in the system
func recreateTunnel(tunnel Tunnel) error {
	var commands []string

	switch strings.ToLower(tunnel.Type) {
	case "sit":
		commands = []string{
			fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", tunnel.ID, tunnel.ServerIPv4, tunnel.ClientIPv4),
			fmt.Sprintf("ip link set %s up", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", tunnel.EndpointLocal, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix1, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix2, tunnel.ID),
		}
		// Add third prefix route if it exists
		if tunnel.DelegatedPrefix3 != "" {
			commands = append(commands, fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix3, tunnel.ID))
		}

	case "gre":
		commands = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", tunnel.ID, tunnel.ServerIPv4, tunnel.ClientIPv4),
			fmt.Sprintf("ip link set %s up", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", tunnel.EndpointLocal, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix1, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix2, tunnel.ID),
		}
		// Add third prefix route if it exists
		if tunnel.DelegatedPrefix3 != "" {
			commands = append(commands, fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix3, tunnel.ID))
		}

	case "wg":
		// WireGuard configuration
		commands = []string{
			fmt.Sprintf("ip link add dev %s type wireguard", tunnel.ID),
			fmt.Sprintf("ip -6 addr add %s dev %s", tunnel.EndpointLocal, tunnel.ID),
			fmt.Sprintf("wg set %s listen-port %d private-key <(echo %s) peer %s allowed-ips %s,%s,%s",
				tunnel.ID, tunnel.ListenPort, tunnel.ServerPrivateKey, tunnel.ClientPublicKey,
				tunnel.EndpointRemote, tunnel.DelegatedPrefix1, tunnel.DelegatedPrefix2),
			fmt.Sprintf("ip link set %s up", tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix1, tunnel.ID),
			fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix2, tunnel.ID),
		}
		// Add third prefix route if it exists
		if tunnel.DelegatedPrefix3 != "" {
			commands = append(commands, fmt.Sprintf("ip -6 route add %s dev %s", tunnel.DelegatedPrefix3, tunnel.ID))
		}

	default:
		return fmt.Errorf("invalid tunnel type: %s", tunnel.Type)
	}

	// Execute commands using sh -c for WireGuard (needed for process substitution)
	for _, cmd := range commands {
		var command *exec.Cmd
		if strings.ToLower(tunnel.Type) == "wg" && strings.Contains(cmd, "<(echo") {
			// WireGuard commands with process substitution need to be executed via bash
			command = exec.Command("bash", "-c", cmd)
		} else {
			parts := strings.Split(cmd, " ")
			command = exec.Command(parts[0], parts[1:]...)
		}
		if err := command.Run(); err != nil {
			return fmt.Errorf("error executing command '%s': %w", cmd, err)
		}
	}

	// Apply security rules
	securityCmd := exec.Command("/etc/tunnelbroker/scripts/tunnel_security.sh")
	if err := securityCmd.Run(); err != nil {
		logger.Printf("TunnelRecovery: Warning: Failed to apply security rules: %v", err)
		// Continue even if security script fails
	}

	return nil
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
