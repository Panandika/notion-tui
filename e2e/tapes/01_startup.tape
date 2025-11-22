# VHS Tape: Startup and Quit
# This tape demonstrates basic application startup and quit functionality

Output 01_startup.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Set environment variables for the test
# Note: Replace with your actual token and database ID for recording
Env NOTION_TUI_NOTION_TOKEN "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
Env NOTION_TUI_DATABASE_ID "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

Type "# Notion TUI - Startup Demo" Sleep 500ms Enter
Sleep 1s

Type "# Starting the application..." Sleep 500ms Enter
Sleep 500ms

# Start the application
Type "./notion-tui" Enter
Sleep 3s

# Wait for UI to fully load
Sleep 2s

# Show the page list loaded
Sleep 2s

# Type 'q' to quit
Type "q"
Sleep 1s

# Done
Type "echo 'Startup and quit test complete!'" Enter
Sleep 1s
