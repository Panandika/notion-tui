package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestNewStyles(t *testing.T) {
	t.Parallel()

	styles := NewStyles()

	assert.NotNil(t, styles)
	assert.Equal(t, darkTheme, styles.Theme())
}

func TestNewStylesWithTheme(t *testing.T) {
	t.Parallel()

	customTheme := Theme{
		Primary:   lipgloss.Color("#FF0000"),
		Secondary: lipgloss.Color("#00FF00"),
		Accent:    lipgloss.Color("#0000FF"),
		Text:      lipgloss.Color("#FFFFFF"),
		Muted:     lipgloss.Color("#888888"),
		Error:     lipgloss.Color("#FF0000"),
		Success:   lipgloss.Color("#00FF00"),
		Warning:   lipgloss.Color("#FFFF00"),
	}

	styles := NewStylesWithTheme(customTheme)

	assert.NotNil(t, styles)
	assert.Equal(t, customTheme, styles.Theme())
}

func TestStyleMethods(t *testing.T) {
	t.Parallel()

	styles := NewStyles()

	tests := []struct {
		name       string
		styleFunc  func() lipgloss.Style
		checkBold  bool
		expectBold bool
	}{
		{
			name:       "TitleStyle returns valid style",
			styleFunc:  styles.TitleStyle,
			checkBold:  true,
			expectBold: true,
		},
		{
			name:      "BoxStyle returns valid style",
			styleFunc: styles.BoxStyle,
			checkBold: false,
		},
		{
			name:      "ListStyle returns valid style",
			styleFunc: styles.ListStyle,
			checkBold: false,
		},
		{
			name:      "StatusStyle returns valid style",
			styleFunc: styles.StatusStyle,
			checkBold: false,
		},
		{
			name:       "ErrorStyle returns valid style",
			styleFunc:  styles.ErrorStyle,
			checkBold:  true,
			expectBold: true,
		},
		{
			name:       "SelectStyle returns valid style",
			styleFunc:  styles.SelectStyle,
			checkBold:  true,
			expectBold: true,
		},
		{
			name:      "MutedStyle returns valid style",
			styleFunc: styles.MutedStyle,
			checkBold: false,
		},
		{
			name:       "SuccessStyle returns valid style",
			styleFunc:  styles.SuccessStyle,
			checkBold:  true,
			expectBold: true,
		},
		{
			name:       "WarningStyle returns valid style",
			styleFunc:  styles.WarningStyle,
			checkBold:  true,
			expectBold: true,
		},
		{
			name:       "HeaderStyle returns valid style",
			styleFunc:  styles.HeaderStyle,
			checkBold:  true,
			expectBold: true,
		},
		{
			name:      "HelpStyle returns valid style",
			styleFunc: styles.HelpStyle,
			checkBold: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			style := tt.styleFunc()
			assert.NotNil(t, style)

			// Test that the style can render text
			rendered := style.Render("test")
			assert.NotEmpty(t, rendered)

			if tt.checkBold {
				assert.Equal(t, tt.expectBold, style.GetBold())
			}
		})
	}
}

func TestStyleRendering(t *testing.T) {
	t.Parallel()

	styles := NewStyles()

	tests := []struct {
		name      string
		styleFunc func() lipgloss.Style
		input     string
	}{
		{
			name:      "TitleStyle renders text",
			styleFunc: styles.TitleStyle,
			input:     "Test Title",
		},
		{
			name:      "ErrorStyle renders text",
			styleFunc: styles.ErrorStyle,
			input:     "Error Message",
		},
		{
			name:      "SelectStyle renders text",
			styleFunc: styles.SelectStyle,
			input:     "Selected Item",
		},
		{
			name:      "StatusStyle renders text",
			styleFunc: styles.StatusStyle,
			input:     "Status: OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			style := tt.styleFunc()
			rendered := style.Render(tt.input)

			// Rendered output should contain the input text
			assert.Contains(t, rendered, tt.input)
		})
	}
}

func TestDarkThemeColors(t *testing.T) {
	t.Parallel()

	assert.Equal(t, lipgloss.Color("#7C3AED"), darkTheme.Primary)
	assert.Equal(t, lipgloss.Color("#374151"), darkTheme.Secondary)
	assert.Equal(t, lipgloss.Color("#10B981"), darkTheme.Accent)
	assert.Equal(t, lipgloss.Color("#F3F4F6"), darkTheme.Text)
	assert.Equal(t, lipgloss.Color("#6B7280"), darkTheme.Muted)
	assert.Equal(t, lipgloss.Color("#EF4444"), darkTheme.Error)
	assert.Equal(t, lipgloss.Color("#10B981"), darkTheme.Success)
	assert.Equal(t, lipgloss.Color("#F59E0B"), darkTheme.Warning)
}

func TestBoxStyleHasBorder(t *testing.T) {
	t.Parallel()

	styles := NewStyles()
	boxStyle := styles.BoxStyle()

	// Verify that box style has border enabled
	border := boxStyle.GetBorderStyle()
	assert.NotEmpty(t, border.Top)
	assert.NotEmpty(t, border.Right)
	assert.NotEmpty(t, border.Bottom)
	assert.NotEmpty(t, border.Left)
}

func TestListStyleHasPadding(t *testing.T) {
	t.Parallel()

	styles := NewStyles()
	listStyle := styles.ListStyle()

	// List style should have left padding
	_, _, _, left := listStyle.GetPadding()
	assert.Greater(t, left, 0)
}
