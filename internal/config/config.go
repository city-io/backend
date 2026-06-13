// Package config loads runtime configuration from the environment. It is
// parsed once at startup and passed through the composition root to the
// components that need it.
package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
)

// Config is the application configuration, populated from environment
// variables.
type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	APIPort     string `env:"API_PORT" envDefault:"8080"`
	JWTSecret   string `env:"JWT_SECRET"`

	// ProdDB and DevDB are populated manually in Load because the underlying
	// environment variables use a _PROD/_DEV suffix that env prefixes cannot
	// express.
	ProdDB DatabaseConfig `env:"-"`
	DevDB  DatabaseConfig `env:"-"`
}

// DatabaseConfig holds the connection settings for a single PostgreSQL
// database.
type DatabaseConfig struct {
	Host     string
	Database string
	Username string
	Password string
	Port     string
}

// Load parses the environment into a Config.
func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}

	cfg.ProdDB = DatabaseConfig{
		Host:     os.Getenv("PSQL_HOST_PROD"),
		Database: os.Getenv("PSQL_DATABASE_PROD"),
		Username: os.Getenv("PSQL_USERNAME_PROD"),
		Password: os.Getenv("PSQL_PASSWORD_PROD"),
		Port:     getOr("PSQL_PORT_PROD", "5432"),
	}
	cfg.DevDB = DatabaseConfig{
		Host:     os.Getenv("PSQL_HOST_DEV"),
		Database: os.Getenv("PSQL_DATABASE_DEV"),
		Username: os.Getenv("PSQL_USERNAME_DEV"),
		Password: os.Getenv("PSQL_PASSWORD_DEV"),
		Port:     getOr("PSQL_PORT_DEV", "5432"),
	}

	return &cfg, nil
}

// IsProduction reports whether the application is running in the production
// environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Database returns the database settings for the active environment.
func (c *Config) Database() DatabaseConfig {
	if c.IsProduction() {
		return c.ProdDB
	}
	return c.DevDB
}

// DatabaseDSN returns the libpq connection string for the active environment.
func (c *Config) DatabaseDSN() string {
	db := c.Database()
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s",
		db.Host, db.Username, db.Password, db.Database, db.Port)
}

func getOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
