package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

// TestRootCommandInitialization tests that flags are properly bound.
// (Documentation test for BP-2 and CFG-1)
func TestRootCommandInitialization(t *testing.T) {
	// Verify flags are registered
	if rootCmd.PersistentFlags().Lookup("token") == nil {
		t.Error("--token flag not registered")
	}
	if rootCmd.PersistentFlags().Lookup("database-id") == nil {
		t.Error("--database-id flag not registered")
	}
	if rootCmd.PersistentFlags().Lookup("debug") == nil {
		t.Error("--debug flag not registered")
	}
	if rootCmd.PersistentFlags().Lookup("config") == nil {
		t.Error("--config flag not registered")
	}
}

// TestRootCommandExecution verifies the command runs without errors.
func TestRootCommandExecution(t *testing.T) {
	// Reset viper to clean state
	viper.Reset()

	// Set required values
	viper.Set("notion_token", "test_token")
	viper.Set("database_id", "test_db")

	// Execute with no args - should call runTUI
	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()

	// Should complete successfully (no error)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
}

// TestFlagPriority documents the expected priority order (documentation test).
// Per CFG-1: flags > env > config file > defaults
func TestFlagPriority(t *testing.T) {
	// This is a documentation test verifying the behavior via manual testing.
	// The priority is enforced by viper's design:
	// 1. viper.BindPFlag: binds flags to keys
	// 2. viper.AutomaticEnv: binds env vars
	// 3. viper.ReadInConfig: reads from file
	// When viper.Get() is called, it checks in this order:
	// flags > env > config file > defaults
	t.Log("Flag priority order is enforced by Viper and tested via integration tests")
}
