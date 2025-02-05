package tunnels

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kofany/tunnelbroker/internal/db"
)

// CountActiveTunnelsByUser zwraca liczbę aktywnych tuneli dla danego użytkownika.
func CountActiveTunnelsByUser(userID string) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM tunnels WHERE user_id=$1 AND status='active'"
	err := db.Pool.QueryRow(context.Background(), query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("błąd zliczania tuneli: %w", err)
	}
	return count, nil
}

// InsertTunnel wstawia rekord tunelu do bazy.
func InsertTunnel(tunnel *Tunnel, tx pgx.Tx) error {
	query := `
    INSERT INTO tunnels (id, user_id, type, status, server_ipv4, client_ipv4, endpoint_local, endpoint_remote, delegated_prefix_1, delegated_prefix_2, created_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
		tunnel.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("błąd wstawiania tunelu: %w", err)
	}
	return nil
}

// CreateTunnelWithTransaction tworzy nowy tunel i aktualizuje liczniki użytkownika w ramach jednej transakcji
func CreateTunnelWithTransaction(tunnel *Tunnel) error {
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("błąd rozpoczęcia transakcji: %w", err)
	}
	defer tx.Rollback(context.Background()) // rollback w przypadku błędu

	// Wstaw tunel
	if err := InsertTunnel(tunnel, tx); err != nil {
		return err
	}

	// Zaktualizuj liczniki użytkownika
	if err := UpdateUserCounters(tunnel.UserID, tx); err != nil {
		return err
	}

	// Zatwierdź transakcję
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("błąd zatwierdzania transakcji: %w", err)
	}

	return nil
}

// UpdateClientIPv4 aktualizuje adres IPv4 klienta dla tunelu.
func UpdateClientIPv4(tunnelID string, newClientIPv4 string) error {
	query := "UPDATE tunnels SET client_ipv4=$1 WHERE id=$2"
	_, err := db.Pool.Exec(context.Background(), query, newClientIPv4, tunnelID)
	if err != nil {
		return fmt.Errorf("błąd aktualizacji client_ipv4: %w", err)
	}
	return nil
}

// DeleteTunnel usuwa tunel o danym tunnelID.
func DeleteTunnel(tunnelID string) error {
	query := "DELETE FROM tunnels WHERE id=$1"
	_, err := db.Pool.Exec(context.Background(), query, tunnelID)
	if err != nil {
		return fmt.Errorf("błąd usuwania tunelu: %w", err)
	}
	return nil
}

// UpdateUserCounters aktualizuje liczniki tuneli użytkownika
func UpdateUserCounters(userID string, tx pgx.Tx) error {
	query := `
        UPDATE users 
        SET created_tunnels = created_tunnels + 1,
            active_tunnels = active_tunnels + 1
        WHERE id = $1
    `
	_, err := tx.Exec(context.Background(), query, userID)
	if err != nil {
		return fmt.Errorf("błąd aktualizacji liczników użytkownika: %w", err)
	}
	return nil
}

// DecrementActiveUserTunnels zmniejsza licznik aktywnych tuneli
func DecrementActiveUserTunnels(userID string) error {
	query := `
        UPDATE users 
        SET active_tunnels = active_tunnels - 1
        WHERE id = $1
    `
	_, err := db.Pool.Exec(context.Background(), query, userID)
	if err != nil {
		return fmt.Errorf("błąd aktualizacji licznika aktywnych tuneli: %w", err)
	}
	return nil
}

// ResetUserCreatedTunnels resetuje licznik utworzonych tuneli
func ResetUserCreatedTunnels(userID string) error {
	query := `
        UPDATE users 
        SET created_tunnels = 0
        WHERE id = $1
    `
	_, err := db.Pool.Exec(context.Background(), query, userID)
	if err != nil {
		return fmt.Errorf("błąd resetowania licznika tuneli: %w", err)
	}
	return nil
}
