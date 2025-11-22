# VHS Tape: Edit Mode
# This tape demonstrates entering edit mode and editing a page

Output 06_edit_mode.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Set environment variables
Env NOTION_TUI_NOTION_TOKEN "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
Env NOTION_TUI_DATABASE_ID "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

Type "# Notion TUI - Edit Mode Demo" Sleep 500ms Enter
Sleep 1s

# Start the application
Type "./notion-tui" Enter
Sleep 3s

# Select a page to edit
Type "Down"
Sleep 500ms
Type "Enter"
Sleep 2s

# View page content
Sleep 1s

# Enter edit mode with 'e'
Type "e"
Sleep 1s

# The page should now be in edit mode
Sleep 1s

# Navigate to a block
Type "Down"
Sleep 500ms

# Edit the block (simulated - would require actual text input)
Type "Enter"
Sleep 500ms

# Type some text
Type "Updated text content"
Sleep 1s

# Apply block transformation (Ctrl+1 for Heading 1)
Type@500ms "Ctrl+1"
Sleep 1s

# View the transformation
Sleep 1s

# Save changes with Ctrl+S
Type@500ms "Ctrl+S"
Sleep 2s

# Wait for save confirmation
Sleep 1s

# Exit edit mode with Esc
Type "Escape"
Sleep 1s

# Go back to list
Type "Escape"
Sleep 1s

# Quit
Type "q"
Sleep 1s

Type "echo 'Edit mode test complete!'" Enter
Sleep 1s
