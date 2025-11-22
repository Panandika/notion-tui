# VHS Tape: Search Functionality
# This tape demonstrates the search feature in the sidebar

Output 03_search.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Set environment variables
Env NOTION_TUI_NOTION_TOKEN "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
Env NOTION_TUI_DATABASE_ID "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

Type "# Notion TUI - Search Demo" Sleep 500ms Enter
Sleep 1s

# Start the application
Type "./notion-tui" Enter
Sleep 3s

# Wait for page list to load
Sleep 1s

# Press '/' to focus search
Type "/"
Sleep 500ms

# Type search query (example: "task")
Type "task"
Sleep 1s

# View filtered results
Sleep 2s

# Clear search by deleting characters
Type "Backspace"
Sleep 300ms
Type "Backspace"
Sleep 300ms
Type "Backspace"
Sleep 300ms
Type "Backspace"
Sleep 300ms

# View all results again
Sleep 1s

# Try another search
Type "note"
Sleep 1s
Sleep 2s

# Press Esc to exit search
Type "Escape"
Sleep 1s

# Quit
Type "q"
Sleep 1s

Type "echo 'Search test complete!'" Enter
Sleep 1s
