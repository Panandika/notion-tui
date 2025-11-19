package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color scheme for the application.
type Theme struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Text      lipgloss.Color
	Muted     lipgloss.Color
	Error     lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
}

// darkTheme is the default dark theme for the application.
var darkTheme = Theme{
	Primary:   lipgloss.Color("#7C3AED"),
	Secondary: lipgloss.Color("#374151"),
	Accent:    lipgloss.Color("#10B981"),
	Text:      lipgloss.Color("#F3F4F6"),
	Muted:     lipgloss.Color("#6B7280"),
	Error:     lipgloss.Color("#EF4444"),
	Success:   lipgloss.Color("#10B981"),
	Warning:   lipgloss.Color("#F59E0B"),
}

// Styles holds all lipgloss styles for the application.
type Styles struct {
	theme Theme
}

// NewStyles creates a new Styles instance with the default dark theme.
func NewStyles() *Styles {
	return &Styles{
		theme: darkTheme,
	}
}

// NewStylesWithTheme creates a new Styles instance with a custom theme.
func NewStylesWithTheme(theme Theme) *Styles {
	return &Styles{
		theme: theme,
	}
}

// Theme returns the current theme.
func (s *Styles) Theme() Theme {
	return s.theme
}

// TitleStyle returns the style for titles.
func (s *Styles) TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Primary).
		Bold(true).
		MarginBottom(1)
}

// BoxStyle returns the style for bordered boxes.
func (s *Styles) BoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.theme.Secondary).
		Padding(1, 2)
}

// ListStyle returns the style for list items.
func (s *Styles) ListStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Text).
		PaddingLeft(2)
}

// StatusStyle returns the style for status bar.
func (s *Styles) StatusStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Muted).
		Background(s.theme.Secondary).
		Padding(0, 1)
}

// ErrorStyle returns the style for error messages.
func (s *Styles) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Error).
		Bold(true)
}

// SelectStyle returns the style for selected items.
func (s *Styles) SelectStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Accent).
		Bold(true)
}

// MutedStyle returns the style for muted text.
func (s *Styles) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Muted)
}

// SuccessStyle returns the style for success messages.
func (s *Styles) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Success).
		Bold(true)
}

// WarningStyle returns the style for warning messages.
func (s *Styles) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Warning).
		Bold(true)
}

// HeaderStyle returns the style for headers.
func (s *Styles) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Primary).
		Bold(true).
		Underline(true)
}

// HelpStyle returns the style for help text.
func (s *Styles) HelpStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.theme.Muted).
		Italic(true)
}
