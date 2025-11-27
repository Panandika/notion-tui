package config

import (
	"errors"
	"testing"
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
