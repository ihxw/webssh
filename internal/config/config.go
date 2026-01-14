package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const Version = "1.0.0"

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
	JWTSecret     string `mapstructure:"jwt_secret"`
	EncryptionKey string `mapstructure:"encryption_key"`
}

type SSHConfig struct {
	Timeout               string `mapstructure:"timeout"`
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
	viper.SetDefault("database.path", "./data/webssh.db")
	viper.SetDefault("ssh.timeout", "30s")
	viper.SetDefault("ssh.max_connections_per_user", 10)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "./logs/app.log")

	// Environment variables override
	viper.SetEnvPrefix("WEBSSH")
	viper.AutomaticEnv()

	// Bind specific environment variables
	viper.BindEnv("server.port", "WEBSSH_PORT")
	viper.BindEnv("database.path", "WEBSSH_DB_PATH")
	viper.BindEnv("security.jwt_secret", "WEBSSH_JWT_SECRET")
	viper.BindEnv("security.encryption_key", "WEBSSH_ENCRYPTION_KEY")

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
		config.Security.JWTSecret = os.Getenv("WEBSSH_JWT_SECRET")
		if config.Security.JWTSecret == "" {
			return nil, fmt.Errorf("JWT secret is required (set WEBSSH_JWT_SECRET environment variable)")
		}
	}

	if config.Security.EncryptionKey == "" {
		config.Security.EncryptionKey = os.Getenv("WEBSSH_ENCRYPTION_KEY")
		if config.Security.EncryptionKey == "" {
			return nil, fmt.Errorf("encryption key is required (set WEBSSH_ENCRYPTION_KEY environment variable)")
		}
	}

	// Validate encryption key length (must be 32 bytes for AES-256)
	if len(config.Security.EncryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes for AES-256")
	}

	return &config, nil
}
