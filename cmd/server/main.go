package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ihxw/webssh/internal/config"
	"github.com/ihxw/webssh/internal/database"
	"github.com/ihxw/webssh/internal/handlers"
	"github.com/ihxw/webssh/internal/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.InitDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Set Gin mode
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.Default()

	// Apply middleware
	router.Use(middleware.CORS())
	router.Use(middleware.Logger())

	// Public routes
	authHandler := handlers.NewAuthHandler(db, cfg)
	router.POST("/api/auth/login", authHandler.Login)
	router.POST("/api/auth/logout", authHandler.Logout)

	// WebSocket SSH route (authenticated via one-time ticket in handler)
	sshWSHandler := handlers.NewSSHWebSocketHandler(db, cfg)
	router.GET("/api/ws/ssh/:hostId", sshWSHandler.HandleWebSocket)

	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(cfg.Security.JWTSecret))
	{
		// Auth routes
		protected.GET("/auth/me", authHandler.GetCurrentUser)
		protected.POST("/auth/ws-ticket", authHandler.GetWSTicket)

		// SSH host routes
		sshHostHandler := handlers.NewSSHHostHandler(db, cfg)
		protected.GET("/ssh-hosts", sshHostHandler.GetHosts)
		protected.POST("/ssh-hosts", sshHostHandler.CreateHost)
		protected.GET("/ssh-hosts/:id", sshHostHandler.GetHost)
		protected.PUT("/ssh-hosts/:id", sshHostHandler.UpdateHost)
		protected.DELETE("/ssh-hosts/:id", sshHostHandler.DeleteHost)

		// Connection log routes
		logHandler := handlers.NewConnectionLogHandler(db)
		protected.GET("/connection-logs", logHandler.GetLogs)

		// Admin routes
		admin := protected.Group("/users")
		admin.Use(middleware.AdminMiddleware())
		{
			userHandler := handlers.NewUserHandler(db)
			admin.GET("", userHandler.GetUsers)
			admin.POST("", userHandler.CreateUser)
			admin.PUT("/:id", userHandler.UpdateUser)
			admin.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	// Serve static files (embedded frontend)
	// In production, this will serve the built Vue app
	if cfg.Server.Mode == "release" {
		router.Static("/assets", "./web/dist/assets")
		router.StaticFile("/", "./web/dist/index.html")
		router.NoRoute(func(c *gin.Context) {
			c.File("./web/dist/index.html")
		})
	}

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting WebSSH server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
