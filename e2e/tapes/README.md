# VHS tapes

[VHS](https://github.com/charmbracelet/vhs) tapes that drive the real app in a terminal and record what happens. They serve as end-to-end smoke tests and as demo recordings.

Every tape only reads from Notion. `06_edit_mode` opens the editor and leaves without saving.

## What you need

- A built binary at the repository root: `go build -o notion-tui .`
- A Notion token in `NOTION_TUI_NOTION_TOKEN`. Never write a token into a tape file: tapes are committed.
- For `05_database_switch`: a database ID in `NOTION_TUI_DATABASE_ID`.
- For `03_search` and `06_edit_mode`: a page called "Getting Started" that the token can reach. Every new Notion workspace has one; otherwise change the search text in those tapes.
- For `06_edit_mode`: `vim`.

## Running

From the repository root:

```bash
export NOTION_TUI_NOTION_TOKEN="ntn_..."
export NOTION_TUI_DATABASE_ID="..."   # only needed for tape 05

vhs e2e/tapes/01_startup.tape
```

Each tape writes two files to `e2e/out/` (ignored by git):

- `<name>.gif`: the recording.
- `<name>.ascii`: the screen as text, frame by frame. Useful to check a run without watching the GIF.

A tape fails if an expected screen does not appear in time (`Wait+Screen` lines), for example when the token is wrong or a page is missing.

### With Docker

No local VHS install needed. Build a Linux binary first, since the tape runs inside the container:

```bash
GOOS=linux go build -o notion-tui .

docker run --rm -v "$PWD":/vhs -w /vhs \
  -e NOTION_TUI_NOTION_TOKEN -e NOTION_TUI_DATABASE_ID \
  --entrypoint sh ghcr.io/charmbracelet/vhs:v0.12.1 \
  -c 'apt-get update -qq && apt-get install -y -qq vim >/dev/null && vhs e2e/tapes/06_edit_mode.tape'
```

The `apt-get` step is only needed for `06_edit_mode`.

## Tapes

| Tape | Shows |
|------|-------|
| `01_startup` | Dashboard, key reference (`?`), quit |
| `02_navigation` | Sidebar tree, opening an item, hiding the sidebar, going back |
| `03_search` | Workspace search, opening a result, scrolling |
| `04_command_palette` | Palette, filtering with `/`, running a command |
| `05_database_switch` | Database list, row list, opening a row |
| `06_edit_mode` | Editing in `$EDITOR`, leaving without saving |

## Writing tapes

- Keys are commands, not text: `Enter`, `Down 3`, `Ctrl+P`, `Escape`. `Type "Enter"` types the word.
- Wait for the app instead of sleeping where the network is involved: `Wait+Screen@30s /text/`.
- Keep tapes read-only so a recording never changes anyone's workspace.
