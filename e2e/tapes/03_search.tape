# VHS tape: Search
# Searches the workspace, opens a result and scrolls it.
#
# Run from the repository root, with the token in the environment:
#   NOTION_TUI_NOTION_TOKEN=... vhs e2e/tapes/03_search.tape
# See e2e/tapes/README.md. This tape only reads from Notion.

Output e2e/out/03_search.gif
Output e2e/out/03_search.ascii

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Dracula"
Set TypingSpeed 60ms

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

# Scroll the page
Type "jjjj"
Sleep 1s
Type "kkkk"
Sleep 1s

# Back to the results, then to the dashboard
Escape
Sleep 1s
Escape
Sleep 500ms
Escape
Sleep 1s

# Quit
Type "q"
Sleep 1s
