package database

import (
	"database/sql"
	"fmt"

	"github.com/kofany/tunnelbroker/internal/state"
	_ "github.com/mattn/go-sqlite3"
)

// DB reprezentuje połączenie z bazą danych
type DB struct {
	*sql.DB
}

// NewDB tworzy nowe połączenie z bazą danych
func NewDB(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Tworzenie tabel
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tunnels (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			client_ipv4 TEXT NOT NULL,
			server_ipv4 TEXT NOT NULL,
			endpoint_prefix TEXT NOT NULL,
			prefix1 TEXT NOT NULL,
			prefix2 TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			modified_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS prefixes (
			prefix TEXT PRIMARY KEY,
			is_endpoint BOOLEAN NOT NULL,
			description TEXT,
			created_at DATETIME NOT NULL
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &DB{db}, nil
}

// SaveTunnel zapisuje tunel w bazie danych
func (db *DB) SaveTunnel(tunnel state.Tunnel) error {
	query := `
		INSERT OR REPLACE INTO tunnels (
			id, user_id, type, client_ipv4, server_ipv4,
			endpoint_prefix, prefix1, prefix2, status, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		tunnel.ID, tunnel.UserID, tunnel.Type,
		tunnel.ClientIPv4, tunnel.ServerIPv4, tunnel.EndpointPrefix,
		tunnel.Prefix1, tunnel.Prefix2, tunnel.Status,
		tunnel.CreatedAt, tunnel.ModifiedAt)

	if err != nil {
		return fmt.Errorf("failed to save tunnel: %v", err)
	}

	return nil
}

// GetTunnel pobiera tunel z bazy danych
func (db *DB) GetTunnel(id string) (state.Tunnel, error) {
	var tunnel state.Tunnel
	err := db.QueryRow(`
		SELECT id, user_id, type, client_ipv4, server_ipv4,
			   endpoint_prefix, prefix1, prefix2, status, created_at, modified_at
		FROM tunnels
		WHERE id = ?
	`, id).Scan(
		&tunnel.ID, &tunnel.UserID, &tunnel.Type,
		&tunnel.ClientIPv4, &tunnel.ServerIPv4, &tunnel.EndpointPrefix,
		&tunnel.Prefix1, &tunnel.Prefix2, &tunnel.Status,
		&tunnel.CreatedAt, &tunnel.ModifiedAt,
	)
	if err != nil {
		return tunnel, fmt.Errorf("failed to get tunnel: %v", err)
	}
	return tunnel, nil
}

// GetAllTunnels pobiera wszystkie tunele z bazy danych
func (db *DB) GetAllTunnels() ([]state.Tunnel, error) {
	rows, err := db.Query(`
		SELECT id, user_id, type, client_ipv4, server_ipv4,
			   endpoint_prefix, prefix1, prefix2, status, created_at, modified_at
		FROM tunnels
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tunnels []state.Tunnel
	for rows.Next() {
		var t state.Tunnel
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Type,
			&t.ClientIPv4, &t.ServerIPv4, &t.EndpointPrefix,
			&t.Prefix1, &t.Prefix2, &t.Status,
			&t.CreatedAt, &t.ModifiedAt,
		)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

// DeleteTunnel usuwa tunel z bazy danych
func (db *DB) DeleteTunnel(id string) error {
	result, err := db.Exec("DELETE FROM tunnels WHERE id = ?", id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("tunnel not found: %s", id)
	}
	return nil
}

// SavePrefix zapisuje prefiks w bazie danych
func (db *DB) SavePrefix(prefix state.MainPrefix) error {
	_, err := db.Exec(`
		INSERT INTO prefixes (prefix, is_endpoint, description, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(prefix) DO UPDATE SET
			is_endpoint=excluded.is_endpoint,
			description=excluded.description
	`,
		prefix.Prefix, prefix.IsEndpoint, prefix.Description, prefix.CreatedAt,
	)
	return err
}

// GetAllPrefixes pobiera wszystkie prefiksy z bazy danych
func (db *DB) GetAllPrefixes() ([]state.MainPrefix, error) {
	rows, err := db.Query(`
		SELECT prefix, is_endpoint, description, created_at
		FROM prefixes
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefixes []state.MainPrefix
	for rows.Next() {
		var p state.MainPrefix
		p.Allocations = make(map[string]bool)
		err := rows.Scan(&p.Prefix, &p.IsEndpoint, &p.Description, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, p)
	}
	return prefixes, nil
}

// IsPrefixInUse sprawdza czy prefiks jest już używany
func (db *DB) IsPrefixInUse(prefix string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM tunnels 
		WHERE prefix1 = ? OR prefix2 = ? OR endpoint_prefix = ?
	`, prefix, prefix, prefix).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check prefix usage: %v", err)
	}
	return count > 0, nil
}
