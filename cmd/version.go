package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Panandika/notion-tui/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including build date and commit hash.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if --short flag is set
		short, _ := cmd.Flags().GetBool("short")

		if short {
			fmt.Println(version.Short())
		} else {
			fmt.Println(version.Info())
		}
	},
}

func init() {
	// Add --short flag
	versionCmd.Flags().BoolP("short", "s", false, "print short version (version number only)")

	// Register version command with root
	rootCmd.AddCommand(versionCmd)
}
