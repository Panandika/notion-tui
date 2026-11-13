# Notion TUI

A keyboard-driven terminal client for Notion, built with Go and Bubble Tea.

Browse your workspace, read pages, edit them in your own editor, create pages and export Markdown without leaving the terminal.

## Features

- **Browse** - Workspace tree in a sidebar, database page lists, a dashboard to start from
- **Read** - Pages rendered as Markdown in the terminal, including nested content, toggles, callouts, columns and tables
- **Edit** - Open a page in `$VISUAL` / `$EDITOR`; only the parts you changed are written back, so formatting elsewhere is untouched
- **Create** - New pages inside the page you are viewing, written in your editor
- **Export** - Save a page as plain Markdown, from the app or with `notion-tui export`
- **Quick capture** - `notion-tui add "text"` adds a database row or a page straight from the shell
- **Read-only mode** - `--readonly` guarantees nothing in Notion is changed
- **Your keys, your colours** - Rebind actions and pick a theme in the config file
- **Search** - Search the workspace or a single database
- **Cache** - Page content is cached on disk for faster reopening
- **Rate limiting** - Stays within Notion's API limits and waits when the API asks it to

## Installation

### From Source

Requires Go 1.25 or later.

```bash
git clone https://github.com/Panandika/notion-tui.git
cd notion-tui
go build -o notion-tui .

# Optional: move to PATH
sudo mv notion-tui /usr/local/bin/  # Linux/macOS
```

Prebuilt binaries will be attached to [Releases](https://github.com/Panandika/notion-tui/releases) once a version is tagged.

## Quick Start

### 1. Create a token

Open the [Notion developer portal](https://www.notion.so/profile/integrations) and create one of:

- **Internal connection** (recommended to start): acts as a bot and can only reach pages you share with it. Give it read, update and insert content capabilities.
- **Personal access token**: acts as you and can reach everything you can in the chosen workspace.

Copy the token (it starts with `ntn_`).

### 2. Share content (internal connection only)

1. Open a page or database in Notion
2. Click **"..."** (top right) → **"Add connections"**
3. Select your connection

Pages inside a shared page are shared too.

### 3. Configure

**Option A: environment variable**
```bash
export NOTION_TUI_NOTION_TOKEN="ntn_xxxxxxxxxxxxx"
notion-tui
```

**Option B: config file**
```bash
mkdir -p ~/.config/notion-tui
echo 'notion_token: "ntn_xxxxxxxxxxxxx"' > ~/.config/notion-tui/config.yaml
notion-tui
```

**Option C: flags**
```bash
notion-tui --token "ntn_xxx" --database-id "db-id"
```

**Priority:** flags > environment variables > config file > defaults

A database ID is optional. Without one the app starts from the workspace tree. To find a database ID, open the database in a browser: it is the 32-character ID in the URL before `?v=`.

## Usage

Press `?` in the app for the full key reference.

### Everywhere
| Key | Action |
|-----|--------|
| `?` | Show help |
| `Ctrl+P` | Command palette |
| `Ctrl+B` | Show or hide the sidebar |
| `Tab` | Switch focus between sidebar and page |
| `Esc` | Go back |
| `q` / `Ctrl+C` | Quit |

### Sidebar
| Key | Action |
|-----|--------|
| `j` / `↓`, `k` / `↑` | Move |
| `h` / `←`, `l` / `→` | Collapse / expand |
| `Enter` | Open a page or database; expand a parent |
| `r` | Reload the workspace tree (open items and the selection are kept) |

### Page
| Key | Action |
|-----|--------|
| `j` / `↓`, `k` / `↑` | Scroll |
| `e` | Edit the page in your editor |
| `p` | Change properties (database rows) |
| `t` | Check or uncheck to-dos |
| `o` | Follow a link: child page, database, mention or web link |
| `[` / `]` | Back / forward through the pages you read (`Esc` also goes back) |
| `/`, `n`, `N` | Find in page, next match, previous match |
| `D` | Archive the page (asks first; restorable from Notion's trash) |
| `Ctrl+N` | New page inside this page |
| `r` | Reload from Notion |

### Database rows
| Key | Action |
|-----|--------|
| `Enter` | Open the row |
| `n` | New row (asks for a title) |
| `s` | Change sort order: Notion order, title, last edited |
| `/` | Filter rows by title |
| `r` | Reload |

### Command palette
`Ctrl+P` opens it; move with `j` / `k`, filter with `/`, run with `Enter`. Commands: Search Workspace, Switch Database, Refresh, New Page, Export Page, Help, Quit.

## Editing

Pressing `e` on a page:

1. fetches the page's current content from Notion,
2. opens it in `$VISUAL`, or `$EDITOR`, or `vi`,
3. on exit, sends only the changed parts back.

Things to know:

- The file is Notion-flavored Markdown: mostly Markdown, with tags such as `<b>`, `<toggle>`, `<callout>` and `<mention>` for things Markdown cannot express. Leave tags you do not want to change as they are.
- Blocks you did not touch are not rewritten, so their formatting and IDs stay.
- Child pages and databases cannot be deleted by an edit. Notion rejects such a save.
- If a save fails, your text is not lost: the app shows the path of the file holding it.
- Very large pages that Notion returns only partially can be read but not edited.

Set an editor that waits until the file is closed, for example:

```bash
export EDITOR="nvim"
export EDITOR="code --wait"
```

## Properties and to-dos

A database row shows its title and properties above the content. Press `p` to change one:

- **Status** and **select**: pick from the options defined in Notion. A select can also be cleared.
- **Checkbox**: toggled directly.
- **Date**: type `YYYY-MM-DD`, or leave empty to clear. Only a single date is edited; a time or end date on the property is replaced.

Other property types (people, relations, formulas, multi-select and so on) are shown but not editable here.

Press `t` on any page to list its to-dos and check or uncheck them with Space. Each change is sent on its own, without opening the editor.

## Creating pages

`Ctrl+N` (or **New Page** in the palette) opens your editor with a template. The first `# heading` becomes the title. The page is created inside the page you are viewing. Leaving the template unchanged cancels.

On a database's row list, `n` asks for a title and creates a new row.

### From the shell

`notion-tui add` creates a database row, or a page inside another page, without opening the app. The first line is the title and further lines are the content, as Markdown. It prints the new page's URL.

```bash
notion-tui add "Call the dentist"                  # row in the default database
notion-tui add "Idea: offline mode" --db <database-id>
git log -5 --oneline | notion-tui add --page <page-id>   # text from standard input
```

## Exporting

From the app, **Export Page** in the palette writes `<page-title>.md` to the directory the app was started in. Existing files are never overwritten; a numbered name is used instead.

From the shell:

```bash
notion-tui export <page-id> > page.md
notion-tui export <page-id> --output page.md
```

Exports are plain Markdown: Notion-specific tags are converted (toggles and callouts become quotes, mentions become links).

## Configuration

The default configuration path is `~/.config/notion-tui/config.yaml`; `--config` selects another file. See [config.example.yaml](config.example.yaml) for every option.

All options can also be set through environment variables with the `NOTION_TUI_` prefix:

```bash
export NOTION_TUI_NOTION_TOKEN="ntn_xxx"
export NOTION_TUI_DATABASE_ID="db-id"
export NOTION_TUI_DEBUG=true
export NOTION_TUI_CACHE_DIR="~/.cache/notion-tui"
```

### Read-only mode

`--readonly` (or `readonly: true`, or `NOTION_TUI_READONLY=true`) turns off everything that changes Notion: editing, properties, to-dos, new pages and rows, archiving and `notion-tui add`. The keys answer with a notice, and the Notion client itself refuses to send a change, so nothing can slip through. Browsing, search, following links and exporting work as usual.

### Theme

`theme:` (or `--theme`, `NOTION_TUI_THEME`) chooses the colours pages are drawn in: `auto` (the default, follows the terminal's background), `dark`, `light`, `dracula`, `tokyo-night`, `pink`, `ascii` or `notty`.

### Keys

Actions can be given other keys under `keys:`:

```yaml
keys:
  edit: E
  quit: ctrl+q
  find_next: ctrl+g
```

A key is one character, `ctrl+<letter>`, `alt+<character>`, `f1`-`f12`, `tab` or `shift+tab`; put punctuation in quotes (`back: "<"`). Keys for moving, confirming and leaving (arrows, `h j k l`, `g G`, `f b u d`, `ctrl+u`, `ctrl+d`, `ctrl+t`, space, `Enter`, `Esc`, `Ctrl+C`) cannot be reassigned. The app refuses to start if a key would mean two things on the same screen, and names both actions. An action's default key stops working once it has another key. The help screen (`?`) and the hints in the status bar show the keys in effect.

Actions: `help`, `palette`, `sidebar`, `focus`, `quit`, `new_page` (everywhere); `edit`, `properties`, `todos`, `links`, `back`, `forward`, `find`, `find_next`, `find_previous`, `archive`, `reload` (page); `row_new`, `row_sort`, `row_filter`, `row_reload`, `row_more` (database rows); `tree_reload` (sidebar). Defaults are listed in [config.example.yaml](config.example.yaml).

### Multiple databases

List databases under `databases:` in the config file with a name and optional icon, and set `default_database`. **Switch Database** in the palette changes between them.

### Cache

Page content is cached under `~/.cache/notion-tui` for one hour. `r` on a page reloads from Notion and refreshes the cache. Page titles and properties are always fetched live, so the app needs a connection.

## Limitations

- Only status, select, date and checkbox properties can be changed.
- Rows are sorted and filtered locally; Notion's saved views are not used.
- Images are shown as links.
- Columns are shown one after another.

## Development

### Prerequisites

- Go 1.25 or later
- Git
- (Optional) VHS for recording integration tests

### Setup

```bash
# Clone repository
git clone https://github.com/Panandika/notion-tui.git
cd notion-tui

# Install dependencies
go mod download

# Copy environment template
cp .env.example .env
# Edit .env with your Notion token

# Build
go build -o notion-tui .

# Run
./notion-tui
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Quality

```bash
# Format code
go fmt ./...

# Lint
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run
```

### End-to-end tapes

[VHS](https://github.com/charmbracelet/vhs) tapes in `e2e/tapes/` drive the real app and record the result. They only read from Notion and take the token from the environment:

```bash
go build -o notion-tui .
NOTION_TUI_NOTION_TOKEN="ntn_..." vhs e2e/tapes/01_startup.tape
```

See [e2e/tapes/README.md](e2e/tapes/README.md) for requirements and a Docker command.

## Architecture

Notion TUI follows the **Elm Architecture** (Model-View-Update) as implemented by Bubble Tea:

```
AppModel (root orchestrator)
├── Pages (tea.Model implementations)
│   ├── List Page (database page list)
│   ├── Detail Page (page viewer)
│   ├── Edit Page (block editor)
│   ├── Search Page (search interface)
│   └── DBList Page (database selector)
├── Components (reusable UI)
│   ├── Sidebar (page list with search)
│   ├── StatusBar (mode, sync status, help)
│   ├── CommandPalette (Ctrl+P interface)
│   ├── Editor (block editing)
│   └── Viewer (markdown rendering)
└── Services
    ├── Notion Client (rate-limited API wrapper)
    └── Cache (file-based page cache)
```

**Key Principles:**
- Never block in `Update()` - all I/O uses Commands
- Interface-driven design for testability
- Rate limiting at client level
- Responsive caching for offline access

## Project Structure

```
notion-tui/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command
│   └── root_test.go
├── internal/
│   ├── config/            # Configuration management
│   ├── notion/            # Notion API client wrapper
│   ├── cache/             # File-based cache
│   └── ui/                # TUI implementation
│       ├── model.go       # Root app model
│       ├── update.go      # Update logic
│       ├── view.go        # View rendering
│       ├── keys.go        # Key bindings
│       ├── navigation.go  # Page routing
│       ├── components/    # Reusable UI components
│       └── pages/         # Page implementations
├── e2e/                   # End-to-end tests
│   └── tapes/            # VHS tape recordings
├── testdata/              # Test fixtures
├── main.go                # Entry point
├── go.mod                 # Dependencies
├── .goreleaser.yaml       # Release configuration
└── config.example.yaml    # Example configuration
```

## Troubleshooting

### "Invalid token" error

- Verify the token was copied in full (current tokens start with `ntn_`)
- Ensure the integration has access to your database
- Check that you've shared the database with the integration

### Pages not loading

- Confirm the database ID is correct (32-character UUID)
- Check your internet connection
- Try refreshing with `r` key
- Enable debug logging: `NOTION_TUI_DEBUG=true notion-tui`

### Cache issues

- Clear cache: `rm -rf ~/.cache/notion-tui`
- Disable cache: set `cache_dir: ""` in config
- Check permissions on cache directory

### Performance issues

- Rate limiting may slow down large databases
- Try enabling cache for offline browsing
- Consider paginating large result sets

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Quick Start:**
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests
5. Run tests and linting (`go test ./...`, `go vet ./...`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Credits

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [jomei/notionapi](https://github.com/jomei/notionapi) - Notion API client
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management

## Acknowledgments

Special thanks to:
- The [Charm](https://charm.sh) team for their amazing TUI libraries
- The Notion API team for providing a robust API
- All contributors and users of this project

## Support

- **Issues:** [GitHub Issues](https://github.com/Panandika/notion-tui/issues)
- **Discussions:** [GitHub Discussions](https://github.com/Panandika/notion-tui/discussions)
- **Documentation:** This README and [CONTRIBUTING.md](CONTRIBUTING.md)

## Roadmap

Planned next, roughly in order:

- Filter page and database lists

---

**Made with ❤️ by the Notion TUI community**
