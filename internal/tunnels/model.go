package tunnels

import "time"

// Tunnel reprezentuje rekord tunelu w bazie.
type Tunnel struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Type             string    `json:"type"`   // "sit", "gre" lub "wg"
	Status           string    `json:"status"` // np. "active"
	ServerIPv4       string    `json:"server_ipv4"`
	ClientIPv4       string    `json:"client_ipv4"`
	EndpointLocal    string    `json:"endpoint_local"`
	EndpointRemote   string    `json:"endpoint_remote"`
	DelegatedPrefix1 string    `json:"delegated_prefix_1"`
	DelegatedPrefix2 string    `json:"delegated_prefix_2"`
	DelegatedPrefix3 string    `json:"delegated_prefix_3"`
	CreatedAt        time.Time `json:"created_at"`
	// WireGuard specific fields
	ServerPrivateKey string `json:"server_private_key,omitempty"` // WireGuard server private key
	ServerPublicKey  string `json:"server_public_key,omitempty"`  // WireGuard server public key
	ClientPrivateKey string `json:"client_private_key,omitempty"` // WireGuard client private key
	ClientPublicKey  string `json:"client_public_key,omitempty"`  // WireGuard client public key
	ListenPort       int    `json:"listen_port,omitempty"`        // WireGuard listen port (default: 51820)
}

// User reprezentuje użytkownika.
type User struct {
	ID             string `json:"id"`
	CreatedTunnels int    `json:"created_tunnels"`
	ActiveTunnels  int    `json:"active_tunnels"`
}
