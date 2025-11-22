# VHS Tape: Database Switching
# This tape demonstrates switching between multiple databases

Output 05_database_switch.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Set environment variables
# Note: This demo requires multiple databases configured
Env NOTION_TUI_NOTION_TOKEN "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
Env NOTION_TUI_DATABASE_ID "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

Type "# Notion TUI - Database Switching Demo" Sleep 500ms Enter
Sleep 1s

# Start the application
Type "./notion-tui" Enter
Sleep 3s

# View current database pages
Sleep 1s

# Open database selector with Ctrl+D
Type@500ms "Ctrl+D"
Sleep 1s

# The database list should appear
Sleep 1s

# Navigate through databases
Type "Down"
Sleep 500ms
Type "Down"
Sleep 500ms

# Select a different database
Type "Enter"
Sleep 2s

# View the new database's pages
Sleep 2s

# Switch back to another database
Type@500ms "Ctrl+D"
Sleep 1s

Type "Up"
Sleep 500ms

Type "Enter"
Sleep 2s

# View updated page list
Sleep 2s

# Quit
Type "q"
Sleep 1s

Type "echo 'Database switching test complete!'" Enter
Sleep 1s
