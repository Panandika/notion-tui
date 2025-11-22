# VHS Tape: Navigation Workflow
# This tape demonstrates navigating through pages with keyboard controls

Output 02_navigation.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Set environment variables
Env NOTION_TUI_NOTION_TOKEN "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
Env NOTION_TUI_DATABASE_ID "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

Type "# Notion TUI - Navigation Demo" Sleep 500ms Enter
Sleep 1s

# Start the application
Type "./notion-tui" Enter
Sleep 3s

# Navigate down with 'j' key (Vim-style)
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms

# Navigate up with 'k' key
Type "k"
Sleep 500ms

# Select page with Enter
Type "Enter"
Sleep 2s

# View the page detail
Sleep 2s

# Go back with Esc
Type "Escape"
Sleep 1s

# Navigate to another page
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms

# Select again
Type "Enter"
Sleep 2s

# Go back
Type "Escape"
Sleep 1s

# Quit
Type "q"
Sleep 1s

Type "echo 'Navigation test complete!'" Enter
Sleep 1s
