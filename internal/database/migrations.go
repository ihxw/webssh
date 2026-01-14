package database

import (
	"fmt"
	"log"

	"github.com/ihxw/webssh/internal/models"
	"gorm.io/gorm"
)

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Auto migrate all models
	err := db.AutoMigrate(
		&models.User{},
		&models.SSHHost{},
		&models.ConnectionLog{},
		&models.SystemConfig{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create default admin user if no users exist
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count == 0 {
		log.Println("Creating default admin user...")
		adminUser := &models.User{
			Username:    "admin",
			Email:       "admin@localhost",
			DisplayName: "Administrator",
			Role:        "admin",
			Status:      "active",
		}
		// Set password to "admin123" - should be changed on first login
		if err := adminUser.SetPassword("admin123"); err != nil {
			return fmt.Errorf("failed to set admin password: %w", err)
		}
		if err := db.Create(adminUser).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		log.Println("Default admin user created (username: admin, password: admin123)")
	}

	log.Println("Database migrations completed successfully")
	return nil
}
