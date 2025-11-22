package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// LayoutSidebarMainInput contains parameters for LayoutSidebarMain.
type LayoutSidebarMainInput struct {
	Sidebar      string
	Main         string
	SidebarWidth int
	TotalWidth   int
	TotalHeight  int
}

// LayoutSidebarMain creates a horizontal layout with a sidebar on the left and main content on the right.
// The sidebar has a border on the right side. Returns empty string if dimensions are invalid.
func LayoutSidebarMain(input LayoutSidebarMainInput) string {
	// Validate dimensions
	if input.TotalWidth <= 0 || input.TotalHeight <= 0 || input.SidebarWidth <= 0 {
		return ""
	}

	if input.SidebarWidth >= input.TotalWidth {
		return ""
	}

	sidebarStyle := lipgloss.NewStyle().
		Width(input.SidebarWidth).
		Height(input.TotalHeight).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(darkTheme.Secondary)

	mainWidth := input.TotalWidth - input.SidebarWidth - 1 // -1 for border
	if mainWidth <= 0 {
		mainWidth = 1
	}

	mainStyle := lipgloss.NewStyle().
		Width(mainWidth).
		Height(input.TotalHeight)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarStyle.Render(input.Sidebar),
		mainStyle.Render(input.Main),
	)
}

// LayoutWithStatusBarInput contains parameters for LayoutWithStatusBar.
type LayoutWithStatusBarInput struct {
	Content   string
	StatusBar string
	Height    int
}

// LayoutWithStatusBar creates a vertical layout with content above and a status bar at the bottom.
// Returns empty string if height is invalid.
func LayoutWithStatusBar(input LayoutWithStatusBarInput) string {
	if input.Height <= 0 {
		return ""
	}

	// Reserve 1 line for status bar
	contentHeight := input.Height - 1
	if contentHeight < 0 {
		contentHeight = 0
	}

	contentStyle := lipgloss.NewStyle().
		Height(contentHeight)

	statusStyle := lipgloss.NewStyle().
		Foreground(darkTheme.Muted).
		Background(darkTheme.Secondary).
		Padding(0, 1)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		contentStyle.Render(input.Content),
		statusStyle.Render(input.StatusBar),
	)
}

// LayoutCommandPaletteInput contains parameters for LayoutCommandPalette.
type LayoutCommandPaletteInput struct {
	Background string
	Palette    string
	Width      int
	Height     int
}

// LayoutCommandPalette overlays a command palette in the center of the screen.
// The background is dimmed with semi-transparent characters.
// Returns the palette centered if dimensions are valid, otherwise returns the palette as-is.
func LayoutCommandPalette(input LayoutCommandPaletteInput) string {
	if input.Width <= 0 || input.Height <= 0 {
		return input.Palette
	}

	// Create semi-transparent overlay effect
	overlay := lipgloss.Place(
		input.Width, input.Height,
		lipgloss.Center, lipgloss.Center,
		input.Palette,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("240")),
	)

	return overlay
}

// CenterOverlayInput contains parameters for CenterOverlay.
type CenterOverlayInput struct {
	Overlay string
	Width   int
	Height  int
}

// CenterOverlay centers content in the available space with a semi-transparent background effect.
// Returns the overlay centered if dimensions are valid, otherwise returns the overlay as-is.
func CenterOverlay(input CenterOverlayInput) string {
	if input.Width <= 0 || input.Height <= 0 {
		return input.Overlay
	}

	return lipgloss.Place(
		input.Width, input.Height,
		lipgloss.Center, lipgloss.Center,
		input.Overlay,
		lipgloss.WithWhitespaceChars("▓"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

// LayoutThreeColumnInput contains parameters for LayoutThreeColumn.
type LayoutThreeColumnInput struct {
	Left   string
	Center string
	Right  string
	Widths [3]int
	Height int
}

// LayoutThreeColumn creates a three-column horizontal layout: [Left] | [Center] | [Right].
// Each column is separated by a border. Returns empty string if dimensions are invalid.
func LayoutThreeColumn(input LayoutThreeColumnInput) string {
	// Validate dimensions
	if input.Height <= 0 {
		return ""
	}

	totalRequiredWidth := input.Widths[0] + input.Widths[1] + input.Widths[2]
	if totalRequiredWidth <= 0 {
		return ""
	}

	// Ensure all widths are positive
	for _, w := range input.Widths {
		if w < 0 {
			return ""
		}
	}

	leftStyle := lipgloss.NewStyle().
		Width(input.Widths[0]).
		Height(input.Height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(darkTheme.Secondary)

	centerStyle := lipgloss.NewStyle().
		Width(input.Widths[1]).
		Height(input.Height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(darkTheme.Secondary)

	rightStyle := lipgloss.NewStyle().
		Width(input.Widths[2]).
		Height(input.Height)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(input.Left),
		centerStyle.Render(input.Center),
		rightStyle.Render(input.Right),
	)
}

// LayoutBoxedContentInput contains parameters for LayoutBoxedContent.
type LayoutBoxedContentInput struct {
	Content string
	Title   string
	Width   int
	Height  int
}

// LayoutBoxedContent creates a bordered box with optional title at the top.
// Uses the application's theme for styling. Returns empty string if dimensions are invalid.
func LayoutBoxedContent(input LayoutBoxedContentInput) string {
	if input.Width <= 0 || input.Height <= 0 {
		return ""
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(darkTheme.Secondary).
		Width(input.Width).
		Height(input.Height).
		Padding(1, 2)

	content := input.Content
	if input.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(darkTheme.Primary).
			Bold(true)

		content = titleStyle.Render(input.Title) + "\n\n" + content
	}

	return boxStyle.Render(content)
}

// LayoutSplitVerticalInput contains parameters for LayoutSplitVertical.
type LayoutSplitVerticalInput struct {
	Top    string
	Bottom string
	Height int
	Split  float64 // 0.0 to 1.0, percentage of height for top section
}

// LayoutSplitVertical creates a vertical split layout with configurable split ratio.
// Split should be between 0.0 and 1.0, representing the percentage of height for the top section.
// Returns empty string if height is invalid or split is out of range.
func LayoutSplitVertical(input LayoutSplitVerticalInput) string {
	if input.Height <= 0 {
		return ""
	}

	if input.Split < 0.0 || input.Split > 1.0 {
		return ""
	}

	topHeight := int(float64(input.Height) * input.Split)
	bottomHeight := input.Height - topHeight

	// Ensure at least 1 line for each section if height allows
	if topHeight == 0 && input.Height > 1 {
		topHeight = 1
		bottomHeight = input.Height - 1
	}
	if bottomHeight == 0 && input.Height > 1 {
		bottomHeight = 1
		topHeight = input.Height - 1
	}

	topStyle := lipgloss.NewStyle().
		Height(topHeight).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(darkTheme.Secondary)

	bottomStyle := lipgloss.NewStyle().
		Height(bottomHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		topStyle.Render(input.Top),
		bottomStyle.Render(input.Bottom),
	)
}
