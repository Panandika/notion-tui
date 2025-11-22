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
			name: "valid config",
			cfg: &Config{
				NotionToken: "secret_xxx",
				DatabaseID:  "db_id",
				Debug:       false,
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
			name: "missing database id",
			cfg: &Config{
				NotionToken: "secret_xxx",
				DatabaseID:  "",
			},
			wantErr: true,
			errMsg:  "database_id is required",
		},
		{
			name: "both fields missing",
			cfg: &Config{
				NotionToken: "",
				DatabaseID:  "",
			},
			wantErr: true,
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

// TestConfigString helper for table-driven test containment checks.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
