# VHS tape: Command palette
# Opens the palette, moves, filters by name and runs a command.
#
# Run from the repository root, with the token in the environment:
#   NOTION_TUI_NOTION_TOKEN=... vhs e2e/tapes/04_command_palette.tape
# See e2e/tapes/README.md. This tape only reads from Notion.

Output e2e/out/04_command_palette.gif
Output e2e/out/04_command_palette.ascii

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Dracula"
Set TypingSpeed 60ms

Type "./notion-tui" Enter
Wait+Screen@30s /Welcome to Notion TUI/
Sleep 1s

# Open the palette and move through the commands
Ctrl+P
Wait+Screen@5s /Command Palette/
Sleep 1s
Down 3
Sleep 700ms
Up 3
Sleep 700ms

# Filter by name with "/", then run the match
Type "/"
Type "help"
Sleep 1s
Enter
Sleep 500ms
Enter
Wait+Screen@5s /Keyboard shortcuts/
Sleep 2s
Escape
Sleep 1s

# Quit
Type "q"
Sleep 1s
