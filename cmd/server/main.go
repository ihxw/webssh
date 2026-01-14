package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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
		protected.POST("/auth/change-password", authHandler.ChangePassword)

		// SSH host routes
		sshHostHandler := handlers.NewSSHHostHandler(db, cfg)
		protected.GET("/ssh-hosts", sshHostHandler.List)
		protected.POST("/ssh-hosts", sshHostHandler.Create)
		protected.PUT("/ssh-hosts/:id", sshHostHandler.Update)
		protected.DELETE("/ssh-hosts/:id", sshHostHandler.Delete)
		protected.GET("/ssh-hosts/:id", sshHostHandler.Get)

		// SFTP routes
		sftpHandler := handlers.NewSftpHandler(db, cfg)
		protected.GET("/sftp/list/:hostId", sftpHandler.List)
		protected.GET("/sftp/download/:hostId", sftpHandler.Download)
		protected.POST("/sftp/upload/:hostId", sftpHandler.Upload)
		protected.DELETE("/sftp/delete/:hostId", sftpHandler.Delete)

		// Connection log routes
		logHandler := handlers.NewConnectionLogHandler(db)
		protected.GET("/connection-logs", logHandler.List)

		// Command template routes
		cmdHandler := handlers.NewCommandTemplateHandler(db)
		protected.GET("/command-templates", cmdHandler.List)
		protected.POST("/command-templates", cmdHandler.Create)
		protected.PUT("/command-templates/:id", cmdHandler.Update)
		protected.DELETE("/command-templates/:id", cmdHandler.Delete)

		// Recording routes
		recHandler := handlers.NewRecordingHandler(db)
		protected.GET("/recordings", recHandler.List)
		protected.GET("/recordings/:id/stream", recHandler.GetStream)
		protected.DELETE("/recordings/:id", recHandler.Delete)

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
	// In development with Vite, this may be ignored as you use port 5173
	// In production or standalone mode, this serves the built Vue app
	router.Static("/assets", "./web/dist/assets")
	router.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	router.StaticFile("/", "./web/dist/index.html")

	router.NoRoute(func(c *gin.Context) {
		// If the request is for an API route, return 404
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}
		// Otherwise serve the index.html for SPA routing
		c.File("./web/dist/index.html")
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting WebSSH server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
