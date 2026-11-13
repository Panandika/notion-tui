# VHS tape: Navigation
# Moves through the workspace tree, opens the first item and goes back.
#
# Run from the repository root, with the token in the environment:
#   NOTION_TUI_NOTION_TOKEN=... vhs e2e/tapes/02_navigation.tape
# See e2e/tapes/README.md. This tape only reads from Notion.

Output e2e/out/02_navigation.gif
Output e2e/out/02_navigation.ascii

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Dracula"
Set TypingSpeed 60ms

Type "./notion-tui" Enter
Wait+Screen@30s /Welcome to Notion TUI/
Sleep 1s

# Give the workspace tree time to load, then focus it and move around
Sleep 3s
Tab
Sleep 500ms
Down 3
Sleep 500ms
Up 3
Sleep 500ms

# Enter on the first item: a page opens in the viewer, a database lists its
# rows, an item with children expands
Enter
Sleep 5s

# Hide and show the sidebar (hiding it also moves focus back to the page)
Ctrl+B
Sleep 1500ms
Ctrl+B
Sleep 1s

# Go back, if there is somewhere to go back to
Escape
Sleep 1s

# Quit
Type "q"
Sleep 1s
