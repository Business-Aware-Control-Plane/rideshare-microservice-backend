package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	URI string
}

// NewPostgresDefaultConfig creates a new PostgreSQL configuration from environment variables
func NewPostgresDefaultConfig() *PostgresConfig {
	uri := os.Getenv("POSTGRES_URI")
	if uri == "" {
		// Fallback for local development or docker-compose
		uri = "postgres://postgres:postgres@localhost:5432/rideshare?sslmode=disable"
	}
	return &PostgresConfig{
		URI: uri,
	}
}

// NewGormDB creates a new Gorm DB instance connected to PostgreSQL
func NewGormDB(cfg *PostgresConfig) (*gorm.DB, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("postgres connection URI is required")
	}

	// GORM database setup
	db, err := gorm.Open(postgres.Open(cfg.URI), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	// Configure connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL via GORM")
	return db, nil
}
