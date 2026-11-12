package pages

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// defaultEditor is used when neither VISUAL nor EDITOR is set.
const defaultEditor = "vi"

// runExternalEditor suspends the TUI, opens path in the user's editor and
// reports the editor's exit through done.
func runExternalEditor(path string, done func(error) tea.Msg) tea.Cmd {
	return tea.ExecProcess(editorCommand(path, os.Getenv), func(err error) tea.Msg {
		return done(err)
	})
}

// editorCommand builds the command that opens path. VISUAL wins over EDITOR,
// as is conventional. The value may carry arguments ("code --wait").
func editorCommand(path string, getenv func(string) string) *exec.Cmd {
	parts := editorParts(getenv)
	args := make([]string, 0, len(parts))
	args = append(args, parts[1:]...)
	args = append(args, path)
	// The program comes from the user's own environment: running their
	// chosen editor is the point.
	return exec.Command(parts[0], args...) //nolint:gosec // G204: see comment above
}

// editorParts returns the editor program and its arguments.
func editorParts(getenv func(string) string) []string {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if parts := strings.Fields(getenv(name)); len(parts) > 0 {
			return parts
		}
	}
	return []string{defaultEditor}
}
