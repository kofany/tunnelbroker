package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var Pool *pgxpool.Pool

func InitDB() error {
	// Ładujemy plik .env (jeśli istnieje)
	if err := godotenv.Load(); err != nil {
		fmt.Println("Nie znaleziono pliku .env, kontynuuję z danymi środowiskowymi")
	}

	host := os.Getenv("SUPABASE_DB_HOST")
	port := os.Getenv("SUPABASE_DB_PORT")
	user := os.Getenv("SUPABASE_DB_USER")
	password := os.Getenv("SUPABASE_DB_PASSWORD")
	dbname := os.Getenv("SUPABASE_DB_NAME")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("błąd parsowania konfiguracji bazy: %w", err)
	}
	config.MaxConns = 10
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("błąd tworzenia puli połączeń: %w", err)
	}
	Pool = pool
	return nil
}

func CloseDB() {
	if Pool != nil {
		Pool.Close()
	}
}
