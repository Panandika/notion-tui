package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/ui"
	"github.com/Panandika/notion-tui/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "notion-tui",
	Short:   "A terminal UI for Notion",
	Version: version.Short(),
	Long: `notion-tui is a keyboard-driven terminal interface for browsing,
editing, and managing your Notion databases and pages.

Configuration can be provided via:
- Command-line flags (highest priority)
- Environment variables (NOTION_TUI_* prefix)
- Configuration file (~/.config/notion-tui/config.yaml)
- Default values (lowest priority)

Example:
  export NOTION_TOKEN="secret_xxx"
  export NOTION_TUI_DATABASE_ID="db_id"
  notion-tui`,
	RunE: runTUI,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Persistent flags (available to all commands)
	rootCmd.PersistentFlags().String(
		"config", "",
		"path to config file (default: ~/.config/notion-tui/config.yaml)",
	)
	rootCmd.PersistentFlags().String(
		"token", "",
		"Notion API token (env: NOTION_TUI_NOTION_TOKEN)",
	)
	rootCmd.PersistentFlags().String(
		"database-id", "",
		"Notion database ID to open (env: NOTION_TUI_DATABASE_ID)",
	)
	rootCmd.PersistentFlags().Bool(
		"debug", false,
		"enable debug logging (env: NOTION_TUI_DEBUG)",
	)

	// Bind flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("notion_token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("database_id", rootCmd.PersistentFlags().Lookup("database-id"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	// Automatically bind environment variables with NOTION_TUI_ prefix
	viper.SetEnvPrefix("NOTION_TUI")
	viper.AutomaticEnv()
}

// initConfig initializes configuration from files and environment.
func initConfig() {
	// If --config flag is set, use that file
	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Default to ~/.config/notion-tui/config.yaml
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting home directory: %v\n", err)
			return
		}

		viper.AddConfigPath(home + "/.config/notion-tui")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.SetConfigType("yaml")

	// Try to read the config file, but don't fail if it doesn't exist
	// (config can come from env vars or flags instead)
	_ = viper.ReadInConfig()
}

// runTUI is the main entry point for the TUI application.
func runTUI(cmd *cobra.Command, args []string) error {
	// Load configuration (validation happens in config.Load)
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create and run the TUI
	model := ui.NewModel(ui.NewModelInput{
		Config: cfg,
		Cache:  nil, // Will use default cache
	})
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}
