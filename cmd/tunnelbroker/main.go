package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kofany/tunnelbroker/internal/config"
	"github.com/kofany/tunnelbroker/internal/db"
	"github.com/kofany/tunnelbroker/internal/middleware"
	"github.com/kofany/tunnelbroker/internal/tunnels"
)

func main() {
	// Ładowanie zmiennych środowiskowych z .env
	if err := godotenv.Load(); err != nil {
		log.Println("Nie znaleziono pliku .env, korzystam z ustawień środowiskowych")
	}

	// Ładowanie konfiguracji
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "cmd/config/config.yaml"
	}
	if err := config.LoadConfig(configPath); err != nil {
		log.Fatalf("Błąd ładowania konfiguracji: %v", err)
	}

	// Inicjalizacja połączenia z bazą
	if err := db.InitDB(); err != nil {
		log.Fatalf("Błąd połączenia z bazą: %v", err)
	}
	defer db.CloseDB()

	// Inicjalizacja routera Gin
	router := gin.Default()

	// Rejestracja endpointów API
	api := router.Group("/api/v1")
	api.Use(middleware.APIKeyAuth())
	{
		api.POST("/tunnels", tunnels.CreateTunnelHandler)
		api.PATCH("/tunnels/:tunnel_id/ip", tunnels.UpdateClientIPHandler)
		api.DELETE("/tunnels/:tunnel_id", tunnels.DeleteTunnelHandler)
	}

	// Nasłuchiwanie tylko na localhost z portem z konfiguracji
	log.Printf("Serwer uruchomiony na %s", config.GlobalConfig.API.Listen)
	if err := router.Run(config.GlobalConfig.API.Listen); err != nil {
		log.Fatalf("Błąd uruchomienia serwera: %v", err)
	}
}
