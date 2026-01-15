package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const Version = "1.2.1"

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Security SecurityConfig `mapstructure:"security"`
	SSH      SSHConfig      `mapstructure:"ssh"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug or release
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type SecurityConfig struct {
	JWTSecret         string `mapstructure:"jwt_secret"`
	EncryptionKey     string `mapstructure:"encryption_key"`
	LoginRateLimit    int    `mapstructure:"login_rate_limit"`
	AccessExpiration  string `mapstructure:"access_expiration"`
	RefreshExpiration string `mapstructure:"refresh_expiration"`
}

type SSHConfig struct {
	Timeout               string `mapstructure:"timeout"`
	IdleTimeout           string `mapstructure:"idle_timeout"`
	MaxConnectionsPerUser int    `mapstructure:"max_connections_per_user"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.path", "./data/termiscope.db")
	viper.SetDefault("ssh.timeout", "30s")
	viper.SetDefault("ssh.idle_timeout", "30m")
	viper.SetDefault("ssh.max_connections_per_user", 10)
	viper.SetDefault("security.login_rate_limit", 20)
	viper.SetDefault("security.access_expiration", "15m")
	viper.SetDefault("security.refresh_expiration", "168h") // 7 days
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "./logs/app.log")

	// Environment variables override
	viper.SetEnvPrefix("TERMISCOPE")
	viper.AutomaticEnv()

	// Bind specific environment variables
	viper.BindEnv("server.port", "TERMISCOPE_PORT")
	viper.BindEnv("database.path", "TERMISCOPE_DB_PATH")
	viper.BindEnv("security.jwt_secret", "TERMISCOPE_JWT_SECRET")
	viper.BindEnv("security.encryption_key", "TERMISCOPE_ENCRYPTION_KEY")

	// Read config file (optional, will use defaults if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Generate secrets if not provided
	if config.Security.JWTSecret == "" {
		config.Security.JWTSecret = os.Getenv("TERMISCOPE_JWT_SECRET")
		if config.Security.JWTSecret == "" {
			return nil, fmt.Errorf("JWT secret is required (set TERMISCOPE_JWT_SECRET environment variable)")
		}
	}

	if config.Security.EncryptionKey == "" {
		config.Security.EncryptionKey = os.Getenv("TERMISCOPE_ENCRYPTION_KEY")
		if config.Security.EncryptionKey == "" {
			return nil, fmt.Errorf("encryption key is required (set TERMISCOPE_ENCRYPTION_KEY environment variable)")
		}
	}

	// Validate encryption key length (must be 32 bytes for AES-256)
	if len(config.Security.EncryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes for AES-256")
	}

	return &config, nil
}

// SaveConfig writes the current configuration back to the config file
func (c *Config) SaveConfig() error {
	viper.Set("server.port", c.Server.Port)
	viper.Set("server.mode", c.Server.Mode)
	viper.Set("database.path", c.Database.Path)
	viper.Set("security.jwt_secret", c.Security.JWTSecret)
	viper.Set("security.encryption_key", c.Security.EncryptionKey)
	viper.Set("security.login_rate_limit", c.Security.LoginRateLimit)
	viper.Set("security.access_expiration", c.Security.AccessExpiration)
	viper.Set("security.refresh_expiration", c.Security.RefreshExpiration)
	viper.Set("ssh.timeout", c.SSH.Timeout)
	viper.Set("ssh.idle_timeout", c.SSH.IdleTimeout)
	viper.Set("ssh.max_connections_per_user", c.SSH.MaxConnectionsPerUser)
	viper.Set("log.level", c.Log.Level)
	viper.Set("log.file", c.Log.File)

	return viper.WriteConfig()
}
