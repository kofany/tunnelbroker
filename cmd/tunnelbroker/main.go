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
	// Load environment variables from .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment settings")
	}

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "cmd/config/config.yaml"
	}
	if err := config.LoadConfig(configPath); err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Initialize database connection
	if err := db.InitDB(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.CloseDB()

	// Initialize Gin router
	router := gin.Default()

	// Register API endpoints
	api := router.Group("/api/v1")
	api.Use(middleware.APIKeyAuth())
	{
		// Tunnel endpoints
		api.GET("/tunnels", tunnels.GetTunnelsHandler)                     // List all tunnels or user tunnels
		api.GET("/tunnels/user/:user_id", tunnels.GetUserTunnelsHandler)   // List tunnels for specific user
		api.GET("/tunnels/:tunnel_id", tunnels.GetTunnelHandler)           // Get specific tunnel details
		api.POST("/tunnels", tunnels.CreateTunnelHandler)                  // Create new tunnel
		api.PATCH("/tunnels/:tunnel_id/ip", tunnels.UpdateClientIPHandler) // Update client IP
		api.DELETE("/tunnels/:tunnel_id", tunnels.DeleteTunnelHandler)     // Delete tunnel
	}

	// Listen only on localhost with port from configuration
	log.Printf("Server started on %s", config.GlobalConfig.API.Listen)
	if err := router.Run(config.GlobalConfig.API.Listen); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
