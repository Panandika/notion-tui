# VHS tape: Databases
# Opens the database list, picks a database, opens a row and goes back. Needs NOTION_TUI_DATABASE_ID.
#
# Run from the repository root, with the token in the environment:
#   NOTION_TUI_NOTION_TOKEN=... vhs e2e/tapes/05_database_switch.tape
# See e2e/tapes/README.md. This tape only reads from Notion.

Output e2e/out/05_database_switch.gif
Output e2e/out/05_database_switch.ascii

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Dracula"
Set TypingSpeed 60ms

Type "./notion-tui" Enter
Wait+Screen@30s /Welcome to Notion TUI/
Sleep 1s

# Switch Database is the second dashboard entry
Down
Sleep 500ms
Enter
Wait+Screen@5s /Select Database/
Sleep 1500ms

# Pick the database: its rows are listed
Enter
Wait+Screen@30s /pages \|/
Sleep 2s

# Move down and open a row
Type "j"
Sleep 500ms
Enter
# The page content is drawn in a box once it has loaded
Wait+Screen@30s /╰─/
Sleep 2s

# Back to the rows
Escape
Sleep 1500ms

# Quit
Type "q"
Sleep 1s
