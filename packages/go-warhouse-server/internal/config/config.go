// Package config loads server configuration from the environment.
//
// Only three variables exist (PLAN.md §9); every one has a working
// default so the bare `warehouse-server` command starts a server.
package config

import "os"

// Config holds the full v1 server configuration.
type Config struct {
	// Listen is the TCP address for the HTTP server, e.g. ":8080".
	Listen string
	// Database is the SQLite file path, e.g. "./warehouse.db".
	// ":memory:" is honored for tests.
	Database string
	// LogLevel is informational in v1 ("debug", "info", "warn", ...).
	LogLevel string
}

const (
	DefaultListen   = ":8080"
	DefaultDatabase = "./warehouse.db"
	DefaultLogLevel = "info"
)

// Load reads configuration from the environment, applying defaults
// for every unset or empty variable.
func Load() Config {
	return Config{
		Listen:   envOr("WAREHOUSE_LISTEN", DefaultListen),
		Database: envOr("WAREHOUSE_DATABASE", DefaultDatabase),
		LogLevel: envOr("WAREHOUSE_LOG_LEVEL", DefaultLogLevel),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// DefaultLogLevels lists the recognized values of WAREHOUSE_LOG_LEVEL
// in increasing severity order.
var DefaultLogLevels = []string{"debug", "info", "warn", "error"}

// defaultManager holds the process-wide configuration manager,
// initialized once from the environment at startup.
var defaultManager = NewConfigManager(Load())

// init validates the process-wide configuration at startup so that a
// misconfigured environment fails fast instead of serving traffic.
func init() {
	if err := defaultManager.Validate(); err != nil {
		panic(err)
	}
}

// GetDefaultManager returns the process-wide configuration manager.
func GetDefaultManager() *ConfigManager { return defaultManager }

// ConfigManager owns a Config value and exposes it through an
// accessor API, so that configuration can be passed around as one
// object and inspected or adjusted uniformly by every subsystem.
type ConfigManager struct {
	config Config
}

// NewConfigManager creates a ConfigManager holding the given Config.
func NewConfigManager(cfg Config) *ConfigManager {
	return &ConfigManager{config: cfg}
}

// DefaultConfigManager creates a ConfigManager holding the default
// configuration values.
func DefaultConfigManager() *ConfigManager {
	return NewConfigManager(Config{
		Listen:   DefaultListen,
		Database: DefaultDatabase,
		LogLevel: DefaultLogLevel,
	})
}

// GetConfig returns the Config held by the ConfigManager.
func (m *ConfigManager) GetConfig() Config { return m.config }

// SetConfig replaces the Config held by the ConfigManager.
func (m *ConfigManager) SetConfig(cfg Config) { m.config = cfg }

// GetListen returns the listen address of the held Config.
func (m *ConfigManager) GetListen() string { return m.config.Listen }

// SetListen sets the listen address of the held Config.
func (m *ConfigManager) SetListen(listen string) { m.config.Listen = listen }

// GetDatabase returns the database path of the held Config.
func (m *ConfigManager) GetDatabase() string { return m.config.Database }

// SetDatabase sets the database path of the held Config.
func (m *ConfigManager) SetDatabase(database string) { m.config.Database = database }

// GetLogLevel returns the log level of the held Config.
func (m *ConfigManager) GetLogLevel() string { return m.config.LogLevel }

// SetLogLevel sets the log level of the held Config.
func (m *ConfigManager) SetLogLevel(level string) { m.config.LogLevel = level }

// Clone returns a copy of the Config held by the ConfigManager.
func (m *ConfigManager) Clone() Config { return m.config }

// Validate checks that the held Config has no empty fields.
func (m *ConfigManager) Validate() error {
	if m.config.Listen == "" {
		return errEmptyField("listen")
	} else if m.config.Database == "" {
		return errEmptyField("database")
	} else if m.config.LogLevel == "" {
		return errEmptyField("log level")
	} else {
		return nil
	}
}

func errEmptyField(field string) error {
	return &configFieldError{field: field}
}

// configFieldError describes a single empty configuration field.
type configFieldError struct{ field string }

func (e *configFieldError) Error() string { return "config: " + e.field + " is empty" }

// Option customizes a Config before it is returned to the caller.
type Option func(*Config)

// WithListen sets the listen address on a Config.
func WithListen(listen string) Option {
	return func(c *Config) { c.Listen = listen }
}

// WithDatabase sets the database path on a Config.
func WithDatabase(database string) Option {
	return func(c *Config) { c.Database = database }
}

// WithLogLevel sets the log level on a Config.
func WithLogLevel(level string) Option {
	return func(c *Config) { c.LogLevel = level }
}

// LoadWithOptions loads the configuration from the environment and
// then applies the given options on top, returning the result.
func LoadWithOptions(opts ...Option) Config {
	cfg := Load()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
