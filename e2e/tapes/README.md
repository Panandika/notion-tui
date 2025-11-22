# VHS Integration Tests

This directory contains [VHS](https://github.com/charmbracelet/vhs) tape files for integration testing and documentation.

## What is VHS?

VHS is a tool for generating terminal GIFs from tape files. It allows you to record, replay, and share terminal sessions as animated images.

## Prerequisites

Install VHS:

```bash
# macOS
brew install vhs

# Go install
go install github.com/charmbracelet/vhs@latest
```

## Running Tapes

### Setup

Before recording tapes, you need to:

1. **Build the application:**
   ```bash
   cd ../..  # Go to project root
   go build -o notion-tui main.go
   ```

2. **Set environment variables:**

   Edit each `.tape` file and replace the placeholder credentials with your actual Notion token and database ID:

   ```tape
   Env NOTION_TUI_NOTION_TOKEN "secret_YOUR_ACTUAL_TOKEN"
   Env NOTION_TUI_DATABASE_ID "your-actual-database-id"
   ```

   **IMPORTANT:** Never commit tape files with real credentials!

3. **Ensure you have test data:**

   Your Notion database should have:
   - Multiple pages (at least 5-10 for navigation demos)
   - Pages with different titles for search testing
   - Pages with editable content for edit mode testing
   - Multiple databases configured for database switching demo

### Recording Tapes

Run VHS on individual tape files:

```bash
# Record a specific tape
vhs 01_startup.tape

# This generates: 01_startup.gif
```

Or record all tapes:

```bash
# Record all tapes
for tape in *.tape; do
  vhs "$tape"
done
```

## Available Tapes

| Tape | Description | Output |
|------|-------------|--------|
| `01_startup.tape` | Basic startup and quit | `01_startup.gif` |
| `02_navigation.tape` | Navigation with j/k/Enter/Esc | `02_navigation.gif` |
| `03_search.tape` | Sidebar search functionality | `03_search.gif` |
| `04_command_palette.tape` | Command palette (Ctrl+P) | `04_command_palette.gif` |
| `05_database_switch.tape` | Switching between databases | `05_database_switch.gif` |
| `06_edit_mode.tape` | Edit mode and saving changes | `06_edit_mode.gif` |

## Customizing Tapes

### Common Settings

```tape
Set Shell "bash"           # Shell to use
Set FontSize 14            # Font size for output
Set Width 1200             # Terminal width
Set Height 800             # Terminal height
Set Theme "Dracula"        # Color theme
```

### Available Themes

- Dracula (default in these tapes)
- Monokai
- Nord
- Catppuccin
- And many more - see [VHS docs](https://github.com/charmbracelet/vhs#themes)

### Timing

Adjust `Sleep` commands to control playback speed:

```tape
Sleep 500ms   # Wait 0.5 seconds
Sleep 1s      # Wait 1 second
Sleep 2s      # Wait 2 seconds
```

## Best Practices

1. **Keep tapes short** - Aim for under 2 minutes each
2. **Focus on one feature** - Each tape should demonstrate a single workflow
3. **Use descriptive comments** - Explain what's happening in the tape
4. **Test before committing** - Always record and verify the output
5. **Protect credentials** - Use placeholders in committed tapes
6. **Consistent styling** - Use the same theme and dimensions across all tapes

## Troubleshooting

### Application doesn't start

- Verify `notion-tui` binary is built and in the correct location
- Check that credentials are valid
- Ensure you're running from the correct directory

### Output looks wrong

- Check terminal dimensions (`Set Width/Height`)
- Try a different theme
- Adjust font size
- Verify your terminal supports true color

### Commands not working

- Check key bindings in the application
- Verify timing with `Sleep` commands
- Some keys may need special syntax (e.g., `Ctrl+P`, `Escape`)

### Slow recording

- Reduce `Sleep` durations
- Skip unnecessary waiting periods
- Use `Type@500ms` for faster typing

## Using GIFs in Documentation

Once generated, you can embed GIFs in documentation:

```markdown
![Startup Demo](e2e/tapes/01_startup.gif)
```

Or link to them in GitHub releases, README, etc.

## CI/CD Integration

To automatically record tapes in CI (requires headless environment):

```yaml
- name: Install VHS
  run: |
    go install github.com/charmbracelet/vhs@latest

- name: Record tapes
  env:
    NOTION_TUI_NOTION_TOKEN: ${{ secrets.NOTION_TOKEN }}
    NOTION_TUI_DATABASE_ID: ${{ secrets.DATABASE_ID }}
  run: |
    cd e2e/tapes
    for tape in *.tape; do
      vhs "$tape"
    done
```

**Note:** This requires setting up secrets in GitHub Actions.

## References

- [VHS Documentation](https://github.com/charmbracelet/vhs)
- [VHS Examples](https://github.com/charmbracelet/vhs/tree/main/examples)
- [Charm Community](https://charm.sh/)
