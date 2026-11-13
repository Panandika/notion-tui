package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/notion"
)

const (
	addTimeout = 60 * time.Second
	// addInputLimit bounds what is read from standard input.
	addInputLimit = 1 << 20
)

var addCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Add a page or database row without opening the UI",
	Long: `Add a database row, or a page inside another page, straight from the
shell. The first line is the title; any further lines are the content,
written as Markdown.

The text comes from the arguments, or from standard input when there are
none (or the only argument is "-").

It goes to the database named with --db, the page named with --page, or
else the default database of the config file.

Example:
  notion-tui add "Call the dentist"
  notion-tui add "Idea: offline mode" --db 4169a22b64ff80fba8e7e1a97d150912
  git log -5 --oneline | notion-tui add --page 4169a22b64ff8031953df82854c539fd`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// From here on a failure is about Notion or the input, not about
		// how the command was typed: the usage text would only bury it.
		cmd.SilenceUsage = true

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.ReadOnly {
			return notion.ErrReadOnly
		}
		databaseID, err := cmd.Flags().GetString("db")
		if err != nil {
			return err
		}
		pageID, err := cmd.Flags().GetString("page")
		if err != nil {
			return err
		}
		parent, err := addParent(databaseID, pageID, cfg.GetDatabaseID())
		if err != nil {
			return err
		}
		text, err := addText(args, cmd.InOrStdin())
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), addTimeout)
		defer cancel()

		created, err := notion.NewClient(cfg.NotionToken).CreatePageMarkdown(ctx, parent, addMarkdown(text))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), created.URL)
		return err
	},
}

// addParent decides where the new page goes.
func addParent(databaseID, pageID, defaultDatabaseID string) (notion.PageParent, error) {
	switch {
	case databaseID != "" && pageID != "":
		return notion.PageParent{}, errors.New("use --db or --page, not both")
	case pageID != "":
		return notion.PageParent{PageID: pageID}, nil
	case databaseID != "":
		return notion.PageParent{DatabaseID: databaseID}, nil
	case defaultDatabaseID != "":
		return notion.PageParent{DatabaseID: defaultDatabaseID}, nil
	default:
		return notion.PageParent{}, errors.New("nowhere to add to: name a database with --db or a page with --page, or set a default database in the config file")
	}
}

// addText returns what to add: the arguments joined by spaces, or standard
// input when there are no arguments or the only one is "-".
func addText(args []string, stdin io.Reader) (string, error) {
	text := strings.Join(args, " ")
	if len(args) == 0 || (len(args) == 1 && args[0] == "-") {
		if file, isFile := stdin.(*os.File); isFile {
			if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
				return "", errors.New("nothing to add: give the text as an argument or pipe it in")
			}
		}
		input, err := io.ReadAll(io.LimitReader(stdin, addInputLimit+1))
		if err != nil {
			return "", fmt.Errorf("read standard input: %w", err)
		}
		if len(input) > addInputLimit {
			return "", errors.New("standard input is larger than 1 MB")
		}
		text = string(input)
	}
	if strings.TrimSpace(text) == "" {
		return "", errors.New("nothing to add: the text is empty")
	}
	return text, nil
}

// headingMarker is the "# " to "###### " that starts a Markdown heading.
var headingMarker = regexp.MustCompile(`^#{1,6}\s+`)

// addMarkdown turns the text into the Markdown of a new page: its first
// line as the title, the rest as the content.
func addMarkdown(text string) string {
	first, rest, _ := strings.Cut(strings.TrimSpace(notion.NormalizeMarkdown(text)), "\n")
	title := strings.TrimSpace(headingMarker.ReplaceAllString(strings.TrimSpace(first), ""))
	if title == "" {
		title = "Untitled"
	}
	markdown := "# " + notion.EscapeMarkdown(title) + "\n"
	if rest = strings.TrimSpace(rest); rest != "" {
		markdown += "\n" + rest + "\n"
	}
	return markdown
}

func init() {
	addCmd.Flags().String("db", "", "database to add a row to")
	addCmd.Flags().String("page", "", "page to add a page inside")
	rootCmd.AddCommand(addCmd)
}
