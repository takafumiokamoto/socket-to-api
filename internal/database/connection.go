package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/sijms/go-ora/v2"
)

type Config struct {
	Host     string
	Port     int
	Service  string
	Username string
	Password string
}

// NewConnection creates a new Oracle database connection using go-ora and sqlx
func NewConnection(cfg Config) (*sqlx.DB, error) {
	// go-ora connection string format:
	// oracle://user:password@host:port/service_name
	connStr := fmt.Sprintf(
		"oracle://%s:%s@%s:%d/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Service,
	)

	db, err := sqlx.Connect("oracle", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
