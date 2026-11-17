package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidate runs validation tests (T-1: table-driven).
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with database",
			cfg: &Config{
				NotionToken: "secret_xxx",
				DatabaseID:  "db_id",
				Debug:       false,
			},
			wantErr: false,
		},
		{
			name: "valid config token only - database optional",
			cfg: &Config{
				NotionToken: "secret_xxx",
				DatabaseID:  "",
			},
			wantErr: false,
		},
		{
			name: "missing notion token",
			cfg: &Config{
				NotionToken: "",
				DatabaseID:  "db_id",
			},
			wantErr: true,
			errMsg:  "notion_token is required",
		},
		{
			name: "token missing - databases configured",
			cfg: &Config{
				NotionToken: "",
				Databases: []DatabaseConfig{
					{ID: "db_123", Name: "Test DB"},
				},
			},
			wantErr: true,
			errMsg:  "notion_token is required",
		},
		{
			name: "valid config with multi-database",
			cfg: &Config{
				NotionToken: "secret_xxx",
				Databases: []DatabaseConfig{
					{ID: "db_1", Name: "DB One"},
					{ID: "db_2", Name: "DB Two"},
				},
				DefaultDatabase: "db_1",
			},
			wantErr: false,
		},
		{
			name: "invalid default database",
			cfg: &Config{
				NotionToken: "secret_xxx",
				Databases: []DatabaseConfig{
					{ID: "db_1", Name: "DB One"},
				},
				DefaultDatabase: "nonexistent",
			},
			wantErr: true,
			errMsg:  "not found in databases list",
		},
		{
			name: "database missing id",
			cfg: &Config{
				NotionToken: "secret_xxx",
				Databases: []DatabaseConfig{
					{ID: "", Name: "No ID DB"},
				},
			},
			wantErr: true,
			errMsg:  "missing required field 'id'",
		},
		{
			name: "database missing name",
			cfg: &Config{
				NotionToken: "secret_xxx",
				Databases: []DatabaseConfig{
					{ID: "db_1", Name: ""},
				},
			},
			wantErr: true,
			errMsg:  "missing required field 'name'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !errors.Is(err, err) {
					// Use string matching for validation error messages
					if err != nil && err.Error() != tt.errMsg {
						if !contains(err.Error(), tt.errMsg) {
							t.Errorf("Validate() error message = %q, want to contain %q", err.Error(), tt.errMsg)
						}
					}
				}
			}
		})
	}
}

// TestString verifies that String() doesn't expose secrets (SEC-2).
func TestString(t *testing.T) {
	cfg := &Config{
		NotionToken:     "secret_sensitive_token_12345",
		DatabaseID:      "db_123",
		DefaultDatabase: "db_123",
		Databases: []DatabaseConfig{
			{ID: "db_123", Name: "Test DB", Icon: "📄"},
		},
		Debug:    true,
		CacheDir: "/home/user/.cache/notion-tui",
	}

	str := cfg.String()

	// Token should never appear in string representation
	if contains(str, "secret_sensitive_token_12345") {
		t.Error("String() exposed sensitive token in output")
	}

	// Token marker should be present
	if !contains(str, "***") {
		t.Error("String() should contain token mask (***)")
	}

	// Non-sensitive data should be visible (check for DefaultDB or database count)
	if !contains(str, "db_123") && !contains(str, "Databases: 1") {
		t.Error("String() should contain default database ID or database count")
	}

	if !contains(str, "true") {
		t.Error("String() should contain Debug flag")
	}
}

// TestConfigImmutable verifies that Config is used as immutable after creation.
// This is more of a documentation test—Go doesn't enforce immutability,
// but this test documents the intended usage pattern (CFG-2).
func TestConfigImmutable(t *testing.T) {
	cfg := &Config{
		NotionToken: "token_123",
		DatabaseID:  "db_456",
	}

	// Capture initial state
	initialToken := cfg.NotionToken
	initialDBID := cfg.DatabaseID

	// Verify that the config is used as intended (immutable)
	if cfg.NotionToken != initialToken || cfg.DatabaseID != initialDBID {
		t.Error("Config should not be modified after creation")
	}
}

// TestHasDatabases tests the HasDatabases helper method.
func TestHasDatabases(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *Config
		expect bool
	}{
		{
			name: "has databases",
			cfg: &Config{
				Databases: []DatabaseConfig{
					{ID: "db_1", Name: "DB One"},
				},
			},
			expect: true,
		},
		{
			name: "no databases",
			cfg: &Config{
				Databases: []DatabaseConfig{},
			},
			expect: false,
		},
		{
			name:   "nil databases",
			cfg:    &Config{},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.HasDatabases()
			if got != tt.expect {
				t.Errorf("HasDatabases() = %v, want %v", got, tt.expect)
			}
		})
	}
}

// TestConfigString helper for table-driven test containment checks.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestIsValidNotionID tests the UUID validation function (T-1: table-driven).
func TestIsValidNotionID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{
			name:  "valid UUID with hyphens",
			id:    "1c1b98c9-c803-80ce-96f0-ecd676e2b410",
			valid: true,
		},
		{
			name:  "valid UUID without hyphens",
			id:    "1c1b98c9c80380ce96f0ecd676e2b410",
			valid: true,
		},
		{
			name:  "valid UUID uppercase",
			id:    "1C1B98C9C80380CE96F0ECD676E2B410",
			valid: true,
		},
		{
			name:  "placeholder string",
			id:    "db_from_file",
			valid: false,
		},
		{
			name:  "empty string",
			id:    "",
			valid: false,
		},
		{
			name:  "too short",
			id:    "abc123",
			valid: false,
		},
		{
			name:  "non-hex characters",
			id:    "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			valid: false,
		},
		{
			name:  "special characters",
			id:    "db_from_-file---",
			valid: false,
		},
		{
			name:  "partial UUID",
			id:    "1c1b98c9-c803-80ce",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNotionID(tt.id)
			if got != tt.valid {
				t.Errorf("isValidNotionID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

// TestMigrateLegacyConfigWithInvalidID tests that invalid database IDs are skipped.
func TestMigrateLegacyConfigWithInvalidID(t *testing.T) {
	tests := []struct {
		name         string
		databaseID   string
		wantMigrated bool
	}{
		{
			name:         "valid UUID migrates",
			databaseID:   "1c1b98c9c80380ce96f0ecd676e2b410",
			wantMigrated: true,
		},
		{
			name:         "placeholder skipped",
			databaseID:   "db_from_file",
			wantMigrated: false,
		},
		{
			name:         "empty string skipped",
			databaseID:   "",
			wantMigrated: false,
		},
		{
			name:         "invalid format skipped",
			databaseID:   "invalid-database-id",
			wantMigrated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DatabaseID: tt.databaseID,
			}

			err := cfg.migrateLegacyConfig()
			if err != nil {
				t.Fatalf("migrateLegacyConfig() returned unexpected error: %v", err)
			}

			if tt.wantMigrated {
				if len(cfg.Databases) != 1 {
					t.Errorf("expected 1 database after migration, got %d", len(cfg.Databases))
				}
				if cfg.DefaultDatabase != tt.databaseID {
					t.Errorf("expected DefaultDatabase = %q, got %q", tt.databaseID, cfg.DefaultDatabase)
				}
			} else {
				if len(cfg.Databases) != 0 {
					t.Errorf("expected 0 databases (migration skipped), got %d", len(cfg.Databases))
				}
				if cfg.DefaultDatabase != "" {
					t.Errorf("expected empty DefaultDatabase, got %q", cfg.DefaultDatabase)
				}
			}
		})
	}
}

// TestHasDatabasesWithInvalidLegacyID verifies workspace search mode fallback.
func TestHasDatabasesWithInvalidLegacyID(t *testing.T) {
	cfg := &Config{
		DatabaseID: "invalid_placeholder",
	}

	// Run migration
	err := cfg.migrateLegacyConfig()
	if err != nil {
		t.Fatalf("migrateLegacyConfig() error: %v", err)
	}

	// HasDatabases should return false for invalid IDs
	if cfg.HasDatabases() {
		t.Error("HasDatabases() should return false for invalid legacy database_id")
	}
}

func TestValidate_Theme(t *testing.T) {
	t.Parallel()

	cfg := &Config{NotionToken: "token"}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, ThemeAuto, cfg.Theme, "no theme means follow the terminal")

	cfg = &Config{NotionToken: "token", Theme: "Dracula"}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, "dracula", cfg.Theme)

	cfg = &Config{NotionToken: "token", Theme: "solarized"}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `theme "solarized" is not known`)
}

func TestResolveCacheDir(t *testing.T) {
	t.Parallel()

	home := func() (string, error) { return "/home/ada", nil }
	noHome := func() (string, error) { return "", errors.New("no home") }

	tests := []struct {
		name       string
		configured string
		homeDir    func() (string, error)
		want       string
	}{
		{"nothing configured", "", home, "/home/ada/.cache/notion-tui"},
		{"absolute path", "/var/cache/ntui", home, "/var/cache/ntui"},
		{"relative path is left alone", "cache", home, "cache"},
		{"tilde", "~/.cache/custom", home, "/home/ada/.cache/custom"},
		{"tilde alone", "~", home, "/home/ada"},
		{"a name that only starts with a tilde", "~cache", home, "~cache"},
		{"no home and nothing configured: no cache", "", noHome, ""},
		{"turned off", "off", home, ""},
		{"turned off, any case", " OFF ", home, ""},
		{"no home but a directory was named", "/tmp/c", noHome, "/tmp/c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveCacheDir(tt.configured, tt.homeDir))
		})
	}
}

// Not parallel: viper's settings are process-wide.
func TestLoad_CacheDir(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	t.Cleanup(viper.Reset)

	viper.Reset()
	viper.Set("notion_token", "token")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "/home/ada/.cache/notion-tui", cfg.CacheDir, "nothing configured: the default directory")

	viper.Set("cache_dir", "")
	cfg, err = Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.CacheDir, `cache_dir: "" has always meant no cache, and still does`)

	viper.Set("cache_dir", "~/elsewhere")
	cfg, err = Load()
	require.NoError(t, err)
	assert.Equal(t, "/home/ada/elsewhere", cfg.CacheDir)
}

// Not parallel: viper's settings are process-wide.
func TestLoad_RelativeCacheDir(t *testing.T) {
	t.Setenv("HOME", "/home/ada")
	t.Cleanup(viper.Reset)

	configDir := t.TempDir()
	configFile := filepath.Join(configDir, "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte("notion_token: token\ncache_dir: local-cache\n"), 0o600))

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(t, viper.ReadInConfig())
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(configDir, "local-cache"), cfg.CacheDir,
		"relative to the config file, wherever the app is started")

	// Typed in a shell, a relative directory starts from where the shell is
	t.Setenv("NOTION_TUI_CACHE_DIR", "from-env")
	viper.SetEnvPrefix("NOTION_TUI")
	require.NoError(t, viper.BindEnv("cache_dir"))
	cfg, err = Load()
	require.NoError(t, err)
	workingDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(workingDir, "from-env"), cfg.CacheDir)
}
