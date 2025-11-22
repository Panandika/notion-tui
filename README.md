# Notion TUI

A powerful, keyboard-driven terminal user interface for Notion, built with Go and Bubble Tea.

Browse, view, edit, and manage your Notion databases and pages without leaving the terminal.

## Features

- **Browse Pages** - Navigate your Notion databases with intuitive keyboard controls
- **View Content** - Render Notion pages with markdown formatting and syntax highlighting
- **Edit Pages** - Full inline editing with block type transformations
- **Search** - Fast fuzzy search across pages in sidebar and dedicated search view
- **Multi-Database Support** - Switch between multiple Notion databases seamlessly
- **Offline Caching** - Local file cache for faster load times and offline access
- **Rate Limiting** - Built-in respect for Notion API limits (3 req/sec)
- **Keyboard-Driven** - Vim-style navigation with no mouse required
- **Beautiful UI** - Terminal styling with Lipgloss and responsive layouts

## Installation

### From Binary (Recommended)

Download the latest release for your platform from the [Releases](https://github.com/Panandika/notion-tui/releases) page.

**Linux/macOS:**
```bash
# Download and extract (replace VERSION and OS/ARCH as needed)
curl -LO https://github.com/Panandika/notion-tui/releases/download/vVERSION/notion-tui-VERSION-OS-ARCH.tar.gz
tar -xzf notion-tui-VERSION-OS-ARCH.tar.gz
sudo mv notion-tui /usr/local/bin/

# Make executable
chmod +x /usr/local/bin/notion-tui
```

**Windows:**
Download the `.zip` file and extract `notion-tui.exe` to a directory in your PATH.

### From Source

Requires Go 1.21 or later.

```bash
git clone https://github.com/Panandika/notion-tui.git
cd notion-tui
go build -o notion-tui main.go

# Optional: Move to PATH
sudo mv notion-tui /usr/local/bin/  # Linux/macOS
# or add to PATH on Windows
```

## Quick Start

### 1. Create a Notion Integration

1. Go to [https://www.notion.so/my-integrations](https://www.notion.so/my-integrations)
2. Click **"+ New integration"**
3. Give it a name (e.g., "Notion TUI")
4. Copy the **Integration Token** (starts with `secret_`)

### 2. Share Database with Integration

1. Open your Notion database in the browser
2. Click **"..."** (top right) → **"Add connections"**
3. Select your integration
4. Copy the database ID from the URL:
   - URL format: `https://notion.so/workspace/DATABASE_ID?v=...`
   - The database ID is the 32-character UUID

### 3. Configure Notion TUI

**Option A: Environment Variables** (Quick)
```bash
export NOTION_TUI_NOTION_TOKEN="secret_xxxxxxxxxxxxx"
export NOTION_TUI_DATABASE_ID="your-database-id"
notion-tui
```

**Option B: Configuration File** (Recommended)
```bash
# Create config directory
mkdir -p ~/.config/notion-tui

# Create config file
cat > ~/.config/notion-tui/config.yaml << EOF
notion_token: "secret_xxxxxxxxxxxxx"
databases:
  - id: "database-id-1"
    name: "My Tasks"
    icon: "✅"
  - id: "database-id-2"
    name: "Notes"
    icon: "📝"
default_database: "database-id-1"
cache_dir: "~/.cache/notion-tui"
debug: false
EOF

# Launch
notion-tui
```

**Option C: Command-Line Flags**
```bash
notion-tui --token "secret_xxx" --database-id "db-id"
```

**Configuration Priority:** Flags > Environment Variables > Config File > Defaults

## Usage

### Keyboard Shortcuts

#### Navigation
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `h` / `←` | Move left |
| `l` / `→` | Move right |
| `Enter` | Select/Open item |
| `Esc` | Go back / Cancel |
| `q` / `Ctrl+C` | Quit application |

#### Search & Commands
| Key | Action |
|-----|--------|
| `/` | Focus sidebar search |
| `Ctrl+P` | Open command palette |
| `Ctrl+D` | Switch database |

#### Page View
| Key | Action |
|-----|--------|
| `e` | Enter edit mode |
| `r` | Refresh page from API |
| `m` | Load more blocks (pagination) |

#### Edit Mode
| Key | Action |
|-----|--------|
| `Ctrl+S` | Save changes |
| `Ctrl+R` | Discard changes and refresh |
| `Esc` | Cancel editing |

**Block Type Transformations:**
| Key | Action |
|-----|--------|
| `Ctrl+1` | Convert to Heading 1 |
| `Ctrl+2` | Convert to Heading 2 |
| `Ctrl+3` | Convert to Heading 3 |
| `Ctrl+P` | Convert to Paragraph |
| `Ctrl+L` | Convert to Bulleted List |
| `Ctrl+O` | Convert to Numbered List |
| `Ctrl+Q` | Convert to Quote |
| `Ctrl+K` | Convert to Code Block |

#### Help
| Key | Action |
|-----|--------|
| `?` | Toggle help screen |

## Configuration

### Configuration File

The default configuration path is `~/.config/notion-tui/config.yaml`. You can specify a custom path with the `--config` flag.

**Full Configuration Example:**
```yaml
# Notion API token (required)
# Get from: https://www.notion.so/my-integrations
notion_token: "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# Database configurations (multiple databases supported)
databases:
  - id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    name: "My Tasks"
    icon: "✅"

  - id: "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
    name: "Notes"
    icon: "📝"

  - id: "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"
    name: "Projects"
    icon: "🚀"

# Default database to show on startup
default_database: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

# Cache directory for offline access (default: ~/.cache/notion-tui)
cache_dir: "~/.cache/notion-tui"

# Enable debug logging (default: false)
# Logs written to debug.log in current directory
debug: false
```

### Environment Variables

All configuration options can be set via environment variables with the `NOTION_TUI_` prefix:

```bash
export NOTION_TUI_NOTION_TOKEN="secret_xxx"
export NOTION_TUI_DATABASE_ID="db-id"
export NOTION_TUI_DEBUG=true
export NOTION_TUI_CACHE_DIR="~/.cache/notion-tui"
```

## Features in Detail

### Offline Caching

Notion TUI caches pages locally for faster load times and offline access:

- **Automatic Caching** - Pages are cached on first view
- **Smart Refresh** - Use `r` to refresh from API when needed
- **Offline Mode** - Browse cached pages without internet connection
- **Cache Location** - Default: `~/.cache/notion-tui`

Cache files are organized by page ID and include metadata for staleness detection.

### Multi-Database Support

Manage multiple Notion databases in one session:

1. **Configure Databases** - Add multiple databases in config file with names and icons
2. **Switch Databases** - Press `Ctrl+D` to open database selector
3. **Default Database** - Set `default_database` in config for startup default

Each database maintains its own page list and search index.

### Search Functionality

Two search modes for finding pages quickly:

**Sidebar Search (`/`):**
- Fuzzy search in sidebar
- Filters visible page list in real-time
- Fast and non-intrusive

**Dedicated Search View (`Ctrl+P` → Search):**
- Full-screen search interface
- Search across all pages in current database
- Shows match highlights and previews

### Edit Mode

Full inline editing capabilities:

- **Block Editing** - Edit text content of any block
- **Type Transformations** - Convert blocks between types (paragraph, heading, list, etc.)
- **Save/Discard** - `Ctrl+S` to save, `Ctrl+R` to discard
- **Validation** - Client-side validation before sending to API
- **Error Handling** - Clear error messages for API failures

**Supported Block Types:**
- Paragraphs
- Headings (H1, H2, H3)
- Bulleted Lists
- Numbered Lists
- Quotes
- Code Blocks

### Rate Limiting

Notion TUI respects Notion's API rate limits:

- **Limit:** 3 requests per second
- **Implementation:** Token bucket with 2.5 req/sec sustained, burst of 3
- **Behavior:** Automatic queueing and retry on rate limit errors

## Development

### Prerequisites

- Go 1.21 or later
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
go build -o notion-tui main.go

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

### VHS Integration Tests

Integration tests use [VHS](https://github.com/charmbracelet/vhs) to record terminal sessions:

```bash
# Install VHS
go install github.com/charmbracelet/vhs@latest

# Record a tape
cd e2e/tapes
vhs 01_startup.tape

# Output: 01_startup.gif
```

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

- Verify your token starts with `secret_`
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
- **Documentation:** See this README and [CLAUDE.md](CLAUDE.md) for development guidelines

## Roadmap

See [WAVE_2_4_PLAN.md](WAVE_2_4_PLAN.md) for current development plans and upcoming features.

**Upcoming Features:**
- Database property editing
- Page creation and deletion
- Kanban board view
- Offline-first sync
- Plugin system
- Custom themes

---

**Made with ❤️ by the Notion TUI community**
