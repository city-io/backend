// Package config loads runtime configuration from the environment. It is
// parsed once at startup and passed through the composition root to the
// components that need it.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config is the application configuration, populated from environment
// variables.
type Config struct {
	Environment string         `env:"ENVIRONMENT" envDefault:"development"`
	APIPort     string         `env:"API_PORT" envDefault:"8080"`
	JWTSecret   string         `env:"JWT_SECRET"`
	DB          DatabaseConfig `envPrefix:"PSQL_"`
}

// DatabaseConfig holds the connection settings for the PostgreSQL database.
// A single set of values is provided per environment; deployments supply the
// appropriate values rather than the app selecting between baked-in blocks.
type DatabaseConfig struct {
	Host     string `env:"HOST" envDefault:"localhost"`
	Port     string `env:"PORT" envDefault:"5432"`
	Database string `env:"DATABASE"`
	Username string `env:"USERNAME"`
	Password string `env:"PASSWORD"`
}

// Load parses the environment into a Config.
func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// IsProduction reports whether the application is running in the production
// environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// DatabaseDSN returns the libpq connection string for the database.
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s",
		c.DB.Host, c.DB.Username, c.DB.Password, c.DB.Database, c.DB.Port)
}
