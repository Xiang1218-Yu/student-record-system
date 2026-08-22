// Package config handles loading application configuration from environment
// variables and configuration files. It is the single source of truth for
// runtime settings and knows nothing about how those settings are consumed.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all application configuration values.
type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`
	DBType     string `mapstructure:"DB_TYPE"`
	DBDSN      string `mapstructure:"DB_DSN"`
	JWTSecret  string `mapstructure:"JWT_SECRET"`
	BaseURL    string `mapstructure:"BASE_URL"`
}

// Load reads configuration from environment variables (and optionally a
// .env file). It applies documented defaults and validates required values.
func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// Apply documented defaults before reading overrides.
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DB_TYPE", "sqlite")
	viper.SetDefault("DB_DSN", "./course.db")
	viper.SetDefault("BASE_URL", "http://localhost:8080")

	// Read .env if present; its absence is not an error.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	cfg := &Config{
		ServerPort: viper.GetString("SERVER_PORT"),
		DBType:     viper.GetString("DB_TYPE"),
		DBDSN:      viper.GetString("DB_DSN"),
		JWTSecret:  viper.GetString("JWT_SECRET"),
		BaseURL:    viper.GetString("BASE_URL"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set (env var or .env file)")
	}
	return cfg, nil
}
