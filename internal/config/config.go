package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/likimiad/ozon_fintech/internal/logger"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// DatabaseConfig represents the database configuration.
type DatabaseConfig struct {
	Name     string `env:"DB_NAME"             env-required:"true"`
	User     string `env:"DB_USER"             env-required:"true"`
	Password string `env:"DB_PASSWORD"         env-required:"true"`
	Port     string `env:"DB_PORT"             env-required:"true"`
	Host     string `env:"DB_HOST"             env-required:"true"`
}

// RedisConfig represents the Redis configuration.
type RedisConfig struct {
	Address  string `env:"REDIS_ADDRESS"       env-required:"true"`
	Password string `env:"REDIS_PASSWORD"      env-default:""`
	DB       int    `env:"REDIS_DB"            env-default:"0"`
}

// ServerConfig represents the server configuration.
type ServerConfig struct {
	Port string `env:"HTTP_PORT"               env-default:"8080"`
}

// Config aggregates all configuration structures.
type Config struct {
	DatabaseConfig
	RedisConfig
	ServerConfig
}

// GetConfig loads and returns the application configuration.
func GetConfig() *Config {
	defer func(start time.Time) {
		slog.Info("Successfully initialized the config file", "duration", time.Since(start))
	}(time.Now())
	return loadConfig()
}

// loadConfig loads the configuration from the .env file.
func loadConfig() *Config {
	exePath, err := os.Executable()
	if err != nil {
		logger.FatalError("Error getting executable path", err)
	}

	exeDir := filepath.Dir(exePath)

	configPath := filepath.Join(exeDir, ".env")
	if _, err := os.Stat(configPath); err != nil {
		logger.FatalError("Error opening config file", err)
	}

	var cfg Config

	err = cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		logger.FatalError("Error reading config file", err)
	}

	return &cfg
}
