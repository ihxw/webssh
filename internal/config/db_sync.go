package config

import (
	"fmt"
	"strconv"

	"github.com/ihxw/termiscope/internal/models"
	"gorm.io/gorm"
)

// Default settings hardcoded as requested
var defaultSettings = map[string]string{
	"ssh.timeout":                  "30s",
	"ssh.idle_timeout":             "30m",
	"ssh.max_connections_per_user": "10",
	"security.login_rate_limit":    "20",
	"security.access_expiration":   "15m",
	"security.refresh_expiration":  "168h",
}

// SyncConfigFromDB loads settings from DB into config, seeding defaults if missing
func SyncConfigFromDB(db *gorm.DB, cfg *Config) error {
	// Ensure table exists (AutoMigrate should have run, but just in case)
	if !db.Migrator().HasTable(&models.SystemConfig{}) {
		return fmt.Errorf("system_config table not found")
	}

	for key, defaultValue := range defaultSettings {
		var setting models.SystemConfig
		err := db.Where("config_key = ?", key).First(&setting).Error

		if err == gorm.ErrRecordNotFound {
			// Seed default
			setting = models.SystemConfig{
				ConfigKey:   key,
				ConfigValue: defaultValue,
				Description: fmt.Sprintf("System setting for %s", key),
			}
			if err := db.Create(&setting).Error; err != nil {
				return fmt.Errorf("failed to seed setting %s: %w", key, err)
			}
		} else if err != nil {
			return err
		}

		// Update in-memory config
		if err := updateConfigValue(cfg, key, setting.ConfigValue); err != nil {
			return fmt.Errorf("failed to load setting %s: %w", key, err)
		}
	}

	return nil
}

// updateConfigValue parses the string value and updates the Config struct
func updateConfigValue(cfg *Config, key, value string) error {
	var err error
	switch key {
	case "ssh.timeout":
		cfg.SSH.Timeout = value
	case "ssh.idle_timeout":
		cfg.SSH.IdleTimeout = value
	case "ssh.max_connections_per_user":
		cfg.SSH.MaxConnectionsPerUser, err = strconv.Atoi(value)
	case "security.login_rate_limit":
		cfg.Security.LoginRateLimit, err = strconv.Atoi(value)
	case "security.access_expiration":
		cfg.Security.AccessExpiration = value
	case "security.refresh_expiration":
		cfg.Security.RefreshExpiration = value
	}
	return err
}
