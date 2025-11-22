package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner wraps the bubbles spinner component.
type Spinner struct {
	spinner spinner.Model
	message string
}

// NewSpinner creates a new spinner with an optional message.
func NewSpinner(message string) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	return Spinner{
		spinner: s,
		message: message,
	}
}

// Init initializes the spinner.
func (s Spinner) Init() tea.Cmd {
	return s.spinner.Tick
}

// Update handles messages for the spinner.
func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View renders the spinner with its message.
func (s Spinner) View() string {
	if s.message != "" {
		return s.spinner.View() + " " + s.message
	}
	return s.spinner.View()
}

// SetMessage updates the spinner message.
func (s *Spinner) SetMessage(message string) {
	s.message = message
}

// Message returns the current message.
func (s Spinner) Message() string {
	return s.message
}
