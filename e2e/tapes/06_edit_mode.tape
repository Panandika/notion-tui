# VHS tape: Editing
# Opens a page in $EDITOR (vim here) and leaves without saving, so nothing changes.
#
# Run from the repository root, with the token in the environment:
#   NOTION_TUI_NOTION_TOKEN=... vhs e2e/tapes/06_edit_mode.tape
# See e2e/tapes/README.md. This tape only reads from Notion.

Output e2e/out/06_edit_mode.gif
Output e2e/out/06_edit_mode.ascii

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Dracula"
Set TypingSpeed 60ms

# The editor is taken from the environment. This tape drives vim.
Env EDITOR "vim"

Type "./notion-tui" Enter
Wait+Screen@30s /Welcome to Notion TUI/
Sleep 1s
# Find a page through search. "Getting Started" exists in every new Notion
# workspace; change the query if yours has no such page.
Ctrl+P
Sleep 500ms
Enter
Wait+Screen@10s /Search \(Workspace\)/
Type "Getting Started"
Enter
Wait+Screen@30s /Search Results/
Sleep 1s
Enter
# The page content is drawn in a box once it has loaded
Wait+Screen@30s /╰─/
Sleep 2s

# "e" hands the terminal to the editor with the page as Markdown
Type "e"
Wait+Screen@30s /\.md/
Sleep 2s

# Look around, then leave without saving
Type "jjj"
Sleep 1s
Type ":q!"
Enter
Wait+Screen@15s /No changes/
Sleep 2s

# Quit
Type "q"
Sleep 1s
