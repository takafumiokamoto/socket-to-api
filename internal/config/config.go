package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	API      APIConfig      `yaml:"api"`
	Worker   WorkerConfig   `yaml:"worker"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig contains TCP server settings
type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	MaxConnections  int           `yaml:"max_connections"`
	KeepAlive       bool          `yaml:"keep_alive"`
	KeepAlivePeriod time.Duration `yaml:"keep_alive_period"`
}

// DatabaseConfig contains Oracle DB settings
type DatabaseConfig struct {
	ConnectionString string        `yaml:"connection_string"`
	MaxOpenConns     int           `yaml:"max_open_conns"`
	MaxIdleConns     int           `yaml:"max_idle_conns"`
	ConnMaxLifetime  time.Duration `yaml:"conn_max_lifetime"`
	PollInterval     time.Duration `yaml:"poll_interval"`
	LockTimeout      time.Duration `yaml:"lock_timeout"`
	TableName        string        `yaml:"table_name"`
	ResponseTable    string        `yaml:"response_table"`
}

// APIConfig contains external API settings
type APIConfig struct {
	BaseURL            string        `yaml:"base_url"`
	Timeout            time.Duration `yaml:"timeout"`
	MaxRetries         int           `yaml:"max_retries"`
	RetryDelay         time.Duration `yaml:"retry_delay"`
	MaxIdleConns       int           `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost int          `yaml:"max_idle_conns_per_host"`
	TLSInsecureSkip    bool          `yaml:"tls_insecure_skip"`
}

// WorkerConfig contains worker pool settings
type WorkerConfig struct {
	PoolSize      int           `yaml:"pool_size"`
	QueueSize     int           `yaml:"queue_size"`
	ProcessTimeout time.Duration `yaml:"process_timeout"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`       // debug, info, warn, error
	Format     string `yaml:"format"`      // json, console
	OutputPath string `yaml:"output_path"` // stdout, stderr, or file path
}

// Load reads and parses the configuration file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.ConnectionString == "" {
		return fmt.Errorf("database connection string is required")
	}

	if c.API.BaseURL == "" {
		return fmt.Errorf("API base URL is required")
	}

	if c.Worker.PoolSize <= 0 {
		return fmt.Errorf("worker pool size must be positive")
	}

	return nil
}

// Default returns a configuration with sensible defaults
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			MaxConnections:  10000,
			KeepAlive:       true,
			KeepAlivePeriod: 60 * time.Second,
		},
		Database: DatabaseConfig{
			MaxOpenConns:    50,
			MaxIdleConns:    25,
			ConnMaxLifetime: 5 * time.Minute,
			PollInterval:    100 * time.Millisecond,
			LockTimeout:     5 * time.Second,
			TableName:       "unsend_data",
			ResponseTable:   "api_responses",
		},
		API: APIConfig{
			Timeout:             30 * time.Second,
			MaxRetries:          3,
			RetryDelay:          1 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			TLSInsecureSkip:     false,
		},
		Worker: WorkerConfig{
			PoolSize:       100,
			QueueSize:      1000,
			ProcessTimeout: 60 * time.Second,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "stdout",
		},
	}
}
