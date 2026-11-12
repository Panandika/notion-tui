package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jomei/notionapi"
	"github.com/spf13/cobra"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/notion"
)

// exportTimeout bounds how long a single export may take.
const exportTimeout = 2 * time.Minute

var exportCmd = &cobra.Command{
	Use:   "export <page-id>",
	Short: "Print a Notion page as Markdown",
	Long: `Print a Notion page as Markdown, without starting the interactive UI.

The Markdown goes to standard output unless --output names a file.

Example:
  notion-tui export 4169a22b64ff8031953df82854c539fd > page.md
  notion-tui export 4169a22b64ff8031953df82854c539fd --output page.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), exportTimeout)
		defer cancel()

		markdown, truncated, err := exportMarkdown(ctx, notion.NewClient(cfg.NotionToken), args[0])
		if err != nil {
			return err
		}
		if truncated {
			warning := "warning: the page is too large; Notion returned only part of it"
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), warning); err != nil {
				return err
			}
		}
		return writeExportOutput(cmd.OutOrStdout(), output, markdown)
	},
}

// pageExporter is the part of the Notion client an export needs.
type pageExporter interface {
	GetPage(ctx context.Context, id string) (*notionapi.Page, error)
	GetPageMarkdown(ctx context.Context, id string) (*notion.PageMarkdown, error)
}

// exportMarkdown returns "# title" followed by the page body as plain Markdown.
func exportMarkdown(ctx context.Context, client pageExporter, pageID string) (markdown string, truncated bool, err error) {
	page, err := client.GetPage(ctx, pageID)
	if err != nil {
		return "", false, err
	}
	content, err := client.GetPageMarkdown(ctx, pageID)
	if err != nil {
		return "", false, err
	}
	body := notion.RenderableMarkdown(content.Markdown)
	return "# " + notion.PageTitle(page) + "\n\n" + body + "\n", content.Truncated, nil
}

// writeExportOutput writes to path, or to stdout when path is empty.
func writeExportOutput(stdout io.Writer, path, markdown string) error {
	if path == "" {
		_, err := io.WriteString(stdout, markdown)
		return err
	}
	// An export is a normal user document, not a secret.
	if err := os.WriteFile(path, []byte(markdown), 0o644); err != nil { //nolint:gosec // G306: see comment above
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func init() {
	exportCmd.Flags().StringP("output", "o", "", "write to this file instead of standard output (replaces an existing file)")
	rootCmd.AddCommand(exportCmd)
}
