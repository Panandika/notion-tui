package pages

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
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

// openInBrowser hands a web link to the system's default browser. Only http
// and https links are accepted, so page content cannot make the app launch
// anything else.
func openInBrowser(target string) error {
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("not a web link: %q", target)
	}

	program, args := "xdg-open", []string{parsed.String()}
	switch runtime.GOOS {
	case "darwin":
		program = "open"
	case "windows":
		program, args = "rundll32", []string{"url.dll,FileProtocolHandler", parsed.String()}
	}
	// A fixed program gets the validated link as one argument; no shell is
	// involved, so the link cannot inject a command.
	cmd := exec.Command(program, args...) //nolint:gosec // G204: see comment above
	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap the helper process once it exits; its result does not matter.
	go func() { _ = cmd.Wait() }()
	return nil
}
