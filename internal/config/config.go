package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex {
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
	// ReadOnly turns off everything that changes Notion.
	ReadOnly bool `mapstructure:"readonly"`
	// Theme is the colour style pages are drawn in; see Themes.
	Theme string `mapstructure:"theme"`
	// Keys gives actions other keys than the default: action name -> key.
	// It is checked by the keymap package when the app starts.
	Keys map[string]string `mapstructure:"keys"`
}

// CacheOff as the cache directory turns the cache off.
const CacheOff = "off"

// ThemeAuto follows the terminal's background colour.
const ThemeAuto = "auto"

// Themes lists the values "theme" accepts.
func Themes() []string {
	return []string{ThemeAuto, "dark", "light", "dracula", "tokyo-night", "pink", "ascii", "notty"}
}

// Load reads configuration from viper and validates it.
// Per BP-2 and CFG-1, configuration is validated on startup.
func Load() (*Config, error) {
	var cfg Config

	// Unmarshal viper config into struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// An empty cache_dir used to be how the cache was turned off, and
	// still is: only a setting that is absent means the default directory.
	if viper.IsSet("cache_dir") && strings.TrimSpace(viper.GetString("cache_dir")) == "" {
		cfg.CacheDir = CacheOff
	}
	cfg.CacheDir = resolveCacheDir(cfg.CacheDir, os.UserHomeDir)

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

// resolveCacheDir decides where pages are kept for faster opening and for
// reading without a connection: the configured directory, with a leading
// "~" standing for the home directory, or ~/.cache/notion-tui. It returns
// "" (no cache) when the user turned the cache off with "off", or when no
// directory is configured and the home directory is unknown.
func resolveCacheDir(configured string, homeDir func() (string, error)) string {
	configured = strings.TrimSpace(configured)
	if strings.EqualFold(configured, CacheOff) {
		return ""
	}
	if configured != "" && configured != "~" && !strings.HasPrefix(configured, "~/") {
		return configured
	}
	home, err := homeDir()
	if err != nil || home == "" {
		if configured == "" {
			return ""
		}
		return configured
	}
	if configured == "" {
		return filepath.Join(home, ".cache", "notion-tui")
	}
	return filepath.Join(home, strings.TrimPrefix(configured, "~"))
}

// migrateLegacyConfig converts old single database config to new multi-database format.
//
//nolint:unparam // Returns an error so Load keeps one shape for every config step.
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

	if c.Theme == "" {
		c.Theme = ThemeAuto
	}
	c.Theme = strings.ToLower(c.Theme)
	if !slices.Contains(Themes(), c.Theme) {
		return fmt.Errorf("theme %q is not known (choose one of: %s)", c.Theme, strings.Join(Themes(), ", "))
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
