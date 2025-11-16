package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
// It is immutable after initialization (per CLAUDE.md CFG-2).
type Config struct {
	NotionToken string `mapstructure:"notion_token"`
	DatabaseID  string `mapstructure:"database_id"`
	Debug       bool   `mapstructure:"debug"`
	CacheDir    string `mapstructure:"cache_dir"`
}

// Load reads configuration from viper and validates it.
// Per BP-2 and CFG-1, configuration is validated on startup.
func Load() (*Config, error) {
	var cfg Config

	// Unmarshal viper config into struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks that required configuration values are set.
// Implements CFG-1: fail fast on invalid config.
func (c *Config) Validate() error {
	if c.NotionToken == "" {
		return errors.New("notion_token is required (set via --token flag, NOTION_TUI_NOTION_TOKEN env var, or config file)")
	}

	if c.DatabaseID == "" {
		return errors.New("database_id is required (set via --database-id flag, NOTION_TUI_DATABASE_ID env var, or config file)")
	}

	return nil
}

// String provides a safe string representation without exposing secrets.
// Implements SEC-2: never log secrets.
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{Token: ***, DatabaseID: %s, Debug: %v, CacheDir: %s}",
		c.DatabaseID,
		c.Debug,
		c.CacheDir,
	)
}
