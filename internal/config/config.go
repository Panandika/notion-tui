package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// isValidNotionID checks if the string is a valid Notion ID format.
// Notion IDs are 32 hex characters, optionally with hyphens.
func isValidNotionID(id string) bool {
	if id == "" {
		return false
	}
	// Remove hyphens and check for 32 hex chars
	cleaned := strings.ReplaceAll(id, "-", "")
	if len(cleaned) != 32 {
		return false
	}
	for _, c := range cleaned {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// DatabaseConfig represents a single database configuration.
type DatabaseConfig struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
	Icon string `mapstructure:"icon"` // Optional emoji/icon
}

// Config holds the application configuration.
// It is immutable after initialization (per CLAUDE.md CFG-2).
type Config struct {
	NotionToken     string           `mapstructure:"notion_token"`
	DatabaseID      string           `mapstructure:"database_id"`      // Deprecated: use Databases
	Databases       []DatabaseConfig `mapstructure:"databases"`        // Multiple database support
	DefaultDatabase string           `mapstructure:"default_database"` // Default database ID
	Debug           bool             `mapstructure:"debug"`
	CacheDir        string           `mapstructure:"cache_dir"`
}

// Load reads configuration from viper and validates it.
// Per BP-2 and CFG-1, configuration is validated on startup.
func Load() (*Config, error) {
	var cfg Config

	// Unmarshal viper config into struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Migrate legacy single database config to new format
	if err := cfg.migrateLegacyConfig(); err != nil {
		return nil, fmt.Errorf("migrate config: %w", err)
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// migrateLegacyConfig converts old single database config to new multi-database format.
func (c *Config) migrateLegacyConfig() error {
	// If new format is already set, skip migration
	if len(c.Databases) > 0 {
		// Ensure default database is set
		if c.DefaultDatabase == "" && len(c.Databases) > 0 {
			c.DefaultDatabase = c.Databases[0].ID
		}
		return nil
	}

	// Only migrate if database_id is a valid Notion UUID
	// Invalid IDs (placeholders, malformed values) are silently skipped,
	// allowing the app to fall back to workspace search mode.
	if c.DatabaseID != "" && isValidNotionID(c.DatabaseID) {
		c.Databases = []DatabaseConfig{
			{
				ID:   c.DatabaseID,
				Name: "Default Database",
				Icon: "📄",
			},
		}
		c.DefaultDatabase = c.DatabaseID
	}

	return nil
}

// Validate checks that required configuration values are set.
// Implements CFG-1: fail fast on invalid config.
func (c *Config) Validate() error {
	if c.NotionToken == "" {
		return errors.New("notion_token is required (set via --token flag, NOTION_TUI_NOTION_TOKEN env var, or config file)")
	}

	// Database is now optional - users can use workspace search to discover content
	// If legacy DatabaseID is set but Databases is empty, migrate it (only if valid UUID)
	if len(c.Databases) == 0 && c.DatabaseID != "" && isValidNotionID(c.DatabaseID) {
		c.Databases = []DatabaseConfig{
			{
				ID:   c.DatabaseID,
				Name: "Default Database",
				Icon: "📄",
			},
		}
		c.DefaultDatabase = c.DatabaseID
	}

	// Validate each database config (if any)
	for i, db := range c.Databases {
		if db.ID == "" {
			return fmt.Errorf("database[%d] is missing required field 'id'", i)
		}
		if db.Name == "" {
			return fmt.Errorf("database[%d] is missing required field 'name'", i)
		}
	}

	// Set default database if databases exist
	if len(c.Databases) > 0 {
		if c.DefaultDatabase == "" {
			c.DefaultDatabase = c.Databases[0].ID
		}

		// Verify default database exists in the list
		found := false
		for _, db := range c.Databases {
			if db.ID == c.DefaultDatabase {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("default_database '%s' not found in databases list", c.DefaultDatabase)
		}
	}

	return nil
}

// String provides a safe string representation without exposing secrets.
// Implements SEC-2: never log secrets.
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{Token: ***, Databases: %d, DefaultDB: %s, Debug: %v, CacheDir: %s}",
		len(c.Databases),
		c.DefaultDatabase,
		c.Debug,
		c.CacheDir,
	)
}

// GetDatabase returns the database config for the given ID.
func (c *Config) GetDatabase(id string) *DatabaseConfig {
	for _, db := range c.Databases {
		if db.ID == id {
			return &db
		}
	}
	return nil
}

// GetDefaultDatabase returns the default database config.
func (c *Config) GetDefaultDatabase() *DatabaseConfig {
	return c.GetDatabase(c.DefaultDatabase)
}

// GetDatabaseID returns the current active database ID (for backward compatibility).
func (c *Config) GetDatabaseID() string {
	if c.DefaultDatabase != "" {
		return c.DefaultDatabase
	}
	if len(c.Databases) > 0 {
		return c.Databases[0].ID
	}
	return c.DatabaseID
}

// HasDatabases returns true if any databases are configured.
func (c *Config) HasDatabases() bool {
	return len(c.Databases) > 0
}
