# VHS tape: Startup and help
# Starts the app, shows the dashboard and the key reference, then quits.
#
# Run from the repository root, with the token in the environment:
#   NOTION_TUI_NOTION_TOKEN=... vhs e2e/tapes/01_startup.tape
# See e2e/tapes/README.md. This tape only reads from Notion.

Output e2e/out/01_startup.gif
Output e2e/out/01_startup.ascii

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Dracula"
Set TypingSpeed 60ms

Type "./notion-tui" Enter
Wait+Screen@30s /Welcome to Notion TUI/
Sleep 1s

# Key reference
Type "?"
Wait+Screen@5s /Keyboard shortcuts/
Sleep 3s

# Any key closes it
Escape
Sleep 1s

# Quit
Type "q"
Sleep 1s
