package database

import (
	"fmt"

	"github.com/dustin/articles-backend/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// Set defaults for empty config values
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}

	port := cfg.Port
	if port == "" {
		port = "5432"
	}

	user := cfg.User
	if user == "" {
		user = "postgres"
	}

	dbName := cfg.DBName
	if dbName == "" {
		dbName = "articles"
	}

	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	// Note: empty password is valid for local development

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, cfg.Password, dbName, port, sslMode)

	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
