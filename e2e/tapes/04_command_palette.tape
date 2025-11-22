# VHS Tape: Command Palette
# This tape demonstrates the command palette (Ctrl+P)

Output 04_command_palette.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Set environment variables
Env NOTION_TUI_NOTION_TOKEN "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
Env NOTION_TUI_DATABASE_ID "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

Type "# Notion TUI - Command Palette Demo" Sleep 500ms Enter
Sleep 1s

# Start the application
Type "./notion-tui" Enter
Sleep 3s

# Open command palette with Ctrl+P
Type@500ms "Ctrl+P"
Sleep 1s

# The command palette should appear
Sleep 1s

# Type to search for a command
Type "search"
Sleep 1s

# Navigate through commands
Type "Down"
Sleep 500ms
Type "Down"
Sleep 500ms

# Select a command with Enter
Type "Enter"
Sleep 1s

# The selected command should execute
Sleep 2s

# Close any opened view with Esc
Type "Escape"
Sleep 1s

# Open command palette again
Type@500ms "Ctrl+P"
Sleep 1s

# Try another command
Type "refresh"
Sleep 1s

# Press Esc to close without selecting
Type "Escape"
Sleep 1s

# Quit
Type "q"
Sleep 1s

Type "echo 'Command palette test complete!'" Enter
Sleep 1s
