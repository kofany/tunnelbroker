package state

import "time"

// Tunnel reprezentuje tunel IPv6
type Tunnel struct {
	ID              string    `json:"id"`
	UserID          int64     `json:"user_id"`  // ID użytkownika z panelu
	Type            string    `json:"type"`
	ClientIPv4      string    `json:"client_ipv4"`
	ServerIPv4      string    `json:"server_ipv4"`
	Status          string    `json:"status"`
	EndpointPrefix  string    `json:"endpoint_prefix"`
	Prefix1         string    `json:"prefix1"`
	Prefix2         string    `json:"prefix2"`
	CreatedAt       time.Time `json:"created_at"`
	ModifiedAt      time.Time `json:"modified_at"`
} 