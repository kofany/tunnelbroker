package tunnels

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kofany/tunnelbroker/internal/db"
)

// CountActiveTunnelsByUser returns the number of active tunnels for a given user.
func CountActiveTunnelsByUser(userID string) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM tunnels WHERE user_id=$1 AND status='active'"
	err := db.Pool.QueryRow(context.Background(), query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting tunnels: %w", err)
	}
	return count, nil
}

// InsertTunnel inserts a tunnel record into the database.
func InsertTunnel(tunnel *Tunnel, tx pgx.Tx) error {
	query := `
    INSERT INTO tunnels (id, user_id, type, status, server_ipv4, client_ipv4, endpoint_local, endpoint_remote, delegated_prefix_1, delegated_prefix_2, delegated_prefix_3, created_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `
	tunnel.CreatedAt = time.Now()
	_, err := tx.Exec(context.Background(), query,
		tunnel.ID,
		tunnel.UserID,
		tunnel.Type,
		tunnel.Status,
		tunnel.ServerIPv4,
		tunnel.ClientIPv4,
		tunnel.EndpointLocal,
		tunnel.EndpointRemote,
		tunnel.DelegatedPrefix1,
		tunnel.DelegatedPrefix2,
		tunnel.DelegatedPrefix3,
		tunnel.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("error inserting tunnel: %w", err)
	}
	return nil
}

// CreateTunnelWithTransaction creates a new tunnel and updates user counters within a single transaction
func CreateTunnelWithTransaction(tunnel *Tunnel) error {
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(context.Background()) // rollback on error

	// Insert tunnel
	if err := InsertTunnel(tunnel, tx); err != nil {
		return err
	}

	// Update user counters
	if err := UpdateUserCounters(tunnel.UserID, tx); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// UpdateClientIPv4 updates the client IPv4 address for a tunnel.
func UpdateClientIPv4(tunnelID string, newClientIPv4 string) error {
	query := "UPDATE tunnels SET client_ipv4=$1 WHERE id=$2"
	_, err := db.Pool.Exec(context.Background(), query, newClientIPv4, tunnelID)
	if err != nil {
		return fmt.Errorf("error updating client_ipv4: %w", err)
	}
	return nil
}

// DeleteTunnel deletes a tunnel with the given tunnelID.
func DeleteTunnel(tunnelID string) error {
	query := "DELETE FROM tunnels WHERE id=$1"
	_, err := db.Pool.Exec(context.Background(), query, tunnelID)
	if err != nil {
		return fmt.Errorf("error deleting tunnel: %w", err)
	}
	return nil
}

// UpdateUserCounters updates user tunnel counters
func UpdateUserCounters(userID string, tx pgx.Tx) error {
	query := `
        UPDATE users
        SET created_tunnels = created_tunnels + 1,
            active_tunnels = active_tunnels + 1
        WHERE id = $1
    `
	_, err := tx.Exec(context.Background(), query, userID)
	if err != nil {
		return fmt.Errorf("error updating user counters: %w", err)
	}
	return nil
}

// DecrementActiveUserTunnels decrements the active tunnels counter
func DecrementActiveUserTunnels(userID string) error {
	query := `
        UPDATE users
        SET active_tunnels = active_tunnels - 1
        WHERE id = $1
    `
	_, err := db.Pool.Exec(context.Background(), query, userID)
	if err != nil {
		return fmt.Errorf("error updating active tunnels counter: %w", err)
	}
	return nil
}

// ResetUserCreatedTunnels resets the created tunnels counter
func ResetUserCreatedTunnels(userID string) error {
	query := `
        UPDATE users
        SET created_tunnels = 0
        WHERE id = $1
    `
	_, err := db.Pool.Exec(context.Background(), query, userID)
	if err != nil {
		return fmt.Errorf("error resetting tunnel counter: %w", err)
	}
	return nil
}

// GetAllTunnels returns all tunnels from the database
func GetAllTunnels() ([]Tunnel, error) {
	query := `
		SELECT id, user_id, type, status, server_ipv4, client_ipv4,
		       endpoint_local, endpoint_remote, delegated_prefix_1,
		       delegated_prefix_2, delegated_prefix_3, created_at
		FROM tunnels
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("error retrieving tunnels: %w", err)
	}
	defer rows.Close()

	var tunnels []Tunnel
	for rows.Next() {
		var t Tunnel
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Type, &t.Status, &t.ServerIPv4,
			&t.ClientIPv4, &t.EndpointLocal, &t.EndpointRemote,
			&t.DelegatedPrefix1, &t.DelegatedPrefix2, &t.DelegatedPrefix3, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

// GetUserTunnels returns tunnels for a specific user
func GetUserTunnels(userID string) ([]Tunnel, error) {
	query := `
		SELECT id, user_id, type, status, server_ipv4, client_ipv4,
		       endpoint_local, endpoint_remote, delegated_prefix_1,
		       delegated_prefix_2, delegated_prefix_3, created_at
		FROM tunnels
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving user tunnels: %w", err)
	}
	defer rows.Close()

	var tunnels []Tunnel
	for rows.Next() {
		var t Tunnel
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Type, &t.Status, &t.ServerIPv4,
			&t.ClientIPv4, &t.EndpointLocal, &t.EndpointRemote,
			&t.DelegatedPrefix1, &t.DelegatedPrefix2, &t.DelegatedPrefix3, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

// GetTunnelByID returns a tunnel with the given ID
func GetTunnelByID(tunnelID string) (*Tunnel, error) {
	query := `
		SELECT id, user_id, type, status, server_ipv4, client_ipv4,
		       endpoint_local, endpoint_remote, delegated_prefix_1,
		       delegated_prefix_2, delegated_prefix_3, created_at
		FROM tunnels
		WHERE id = $1
	`
	var tunnel Tunnel
	err := db.Pool.QueryRow(context.Background(), query, tunnelID).Scan(
		&tunnel.ID, &tunnel.UserID, &tunnel.Type, &tunnel.Status,
		&tunnel.ServerIPv4, &tunnel.ClientIPv4, &tunnel.EndpointLocal,
		&tunnel.EndpointRemote, &tunnel.DelegatedPrefix1,
		&tunnel.DelegatedPrefix2, &tunnel.DelegatedPrefix3, &tunnel.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tunnel not found with ID %s", tunnelID)
		}
		return nil, fmt.Errorf("error retrieving tunnel: %w", err)
	}
	return &tunnel, nil
}

// GetUserByID returns a user with the given ID
func GetUserByID(userID string) (*User, error) {
	query := `
		SELECT id, created_tunnels, active_tunnels
		FROM users
		WHERE id = $1
	`
	var user User
	err := db.Pool.QueryRow(context.Background(), query, userID).Scan(
		&user.ID, &user.CreatedTunnels, &user.ActiveTunnels,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found with ID %s", userID)
		}
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}
	return &user, nil
}

// IsPrefixInUse checks if a prefix is already in use by any tunnel
func IsPrefixInUse(prefix string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM tunnels
		WHERE delegated_prefix_1 = $1 OR delegated_prefix_2 = $1 OR delegated_prefix_3 = $1
	`
	var count int
	err := db.Pool.QueryRow(context.Background(), query, prefix).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking prefix usage: %w", err)
	}
	return count > 0, nil
}
