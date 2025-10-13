package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/godror/godror"
	"github.com/okamoto/socket-to-api/internal/config"
	"go.uber.org/zap"
)

// OracleDB represents an Oracle database client
type OracleDB struct {
	db     *sql.DB
	config *config.DatabaseConfig
	logger *zap.Logger
}

// NewOracleDB creates a new Oracle database client
func NewOracleDB(cfg *config.DatabaseConfig, logger *zap.Logger) (*OracleDB, error) {
	db, err := sql.Open("godror", cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established",
		zap.Int("max_open_conns", cfg.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.MaxIdleConns))

	return &OracleDB{
		db:     db,
		config: cfg,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (o *OracleDB) Close() error {
	o.logger.Info("closing database connection")
	return o.db.Close()
}

// GetDB returns the underlying database connection
func (o *OracleDB) GetDB() *sql.DB {
	return o.db
}

// Ping checks if the database is reachable
func (o *OracleDB) Ping(ctx context.Context) error {
	return o.db.PingContext(ctx)
}

// BeginTx starts a new transaction
func (o *OracleDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return o.db.BeginTx(ctx, opts)
}

// Stats returns database statistics
func (o *OracleDB) Stats() sql.DBStats {
	return o.db.Stats()
}

// HealthCheck performs a health check on the database
func (o *OracleDB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := o.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	stats := o.Stats()
	o.logger.Debug("database health check",
		zap.Int("open_connections", stats.OpenConnections),
		zap.Int("in_use", stats.InUse),
		zap.Int("idle", stats.Idle))

	return nil
}
