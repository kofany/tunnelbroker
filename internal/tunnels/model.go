package tunnels

import "time"

// Tunnel reprezentuje rekord tunelu w bazie.
type Tunnel struct {
    ID                string    `json:"id"`
    UserID            string    `json:"user_id"`
    Type              string    `json:"type"` // "sit" lub "gre"
    Status            string    `json:"status"` // np. "active"
    ServerIPv4        string    `json:"server_ipv4"`
    ClientIPv4        string    `json:"client_ipv4"`
    EndpointLocal     string    `json:"endpoint_local"`
    EndpointRemote    string    `json:"endpoint_remote"`
    DelegatedPrefix1  string    `json:"delegated_prefix_1"`
    DelegatedPrefix2  string    `json:"delegated_prefix_2"`
    CreatedAt         time.Time `json:"created_at"`
}

// User reprezentuje użytkownika.
type User struct {
    ID             string `json:"id"`
    CreatedTunnels int    `json:"created_tunnels"`
    ActiveTunnels  int    `json:"active_tunnels"`
}