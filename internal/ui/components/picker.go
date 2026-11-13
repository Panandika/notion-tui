package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PickerItem is one choice in a Picker.
type PickerItem struct {
	Label  string
	Detail string // Shown dimmed after the label, for example a current value
}

// Picker is a small vertical menu: a title, a list of choices and a cursor.
// The page that owns it decides what choosing an item means.
type Picker struct {
	title  string
	items  []PickerItem
	cursor int
}

// NewPicker creates a picker with the cursor on the first item.
func NewPicker(title string, items []PickerItem) Picker {
	return Picker{title: title, items: items}
}

// SetItems replaces the choices, keeping the cursor on the same row when
// that row still exists.
func (p *Picker) SetItems(items []PickerItem) {
	p.items = items
	p.SetCursor(p.cursor)
}

// SetCursor moves the cursor, clamped to the available items.
func (p *Picker) SetCursor(index int) {
	switch {
	case len(p.items) == 0 || index < 0:
		p.cursor = 0
	case index >= len(p.items):
		p.cursor = len(p.items) - 1
	default:
		p.cursor = index
	}
}

// Cursor returns the index of the highlighted item.
func (p Picker) Cursor() int {
	return p.cursor
}

// Len returns the number of items.
func (p Picker) Len() int {
	return len(p.items)
}

// HandleKey moves the cursor for navigation keys and reports whether the
// key was one of them.
func (p *Picker) HandleKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up", "k":
		p.SetCursor(p.cursor - 1)
	case "down", "j":
		p.SetCursor(p.cursor + 1)
	case "home", "g":
		p.SetCursor(0)
	case "end", "G":
		p.SetCursor(len(p.items) - 1)
	default:
		return false
	}
	return true
}

// View renders the picker within width by height, scrolling so the cursor
// stays visible.
func (p Picker) View(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	const chrome = 2 // Title and the blank line under it
	visible := height - chrome
	if visible < 1 {
		visible = 1
	}
	start := 0
	if p.cursor >= visible {
		start = p.cursor - visible + 1
	}
	end := start + visible
	if end > len(p.items) {
		end = len(p.items)
	}

	labelWidth := 0
	for _, item := range p.items {
		if w := lipgloss.Width(item.Label); w > labelWidth {
			labelWidth = w
		}
	}

	title := p.title
	if len(p.items) > visible {
		title = fmt.Sprintf("%s (%d/%d)", p.title, p.cursor+1, len(p.items))
	}
	lines := []string{titleStyle.Render(title), ""}
	for i := start; i < end; i++ {
		item := p.items[i]
		label := item.Label + strings.Repeat(" ", labelWidth-lipgloss.Width(item.Label))
		if i == p.cursor {
			line := "> " + label
			if item.Detail != "" {
				line += "  " + item.Detail
			}
			lines = append(lines, selectedStyle.Render(line))
			continue
		}
		line := "  " + label
		if item.Detail != "" {
			line += "  " + mutedStyle.Render(item.Detail)
		}
		lines = append(lines, line)
	}

	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
