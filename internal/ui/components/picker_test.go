package components

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func pickerKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestPicker_Navigation(t *testing.T) {
	t.Parallel()

	picker := NewPicker("Pick", []PickerItem{{Label: "a"}, {Label: "b"}, {Label: "c"}})

	assert.True(t, picker.HandleKey(pickerKey('j')))
	assert.Equal(t, 1, picker.Cursor())
	picker.HandleKey(pickerKey('G'))
	assert.Equal(t, 2, picker.Cursor())
	picker.HandleKey(pickerKey('j'))
	assert.Equal(t, 2, picker.Cursor(), "cursor stops at the last item")
	picker.HandleKey(pickerKey('g'))
	picker.HandleKey(pickerKey('k'))
	assert.Equal(t, 0, picker.Cursor(), "cursor stops at the first item")

	assert.False(t, picker.HandleKey(pickerKey('x')), "other keys are left to the owner")
	assert.False(t, picker.HandleKey(tea.KeyMsg{Type: tea.KeyEnter}))
}

func TestPicker_SetItemsKeepsCursorInRange(t *testing.T) {
	t.Parallel()

	picker := NewPicker("Pick", []PickerItem{{Label: "a"}, {Label: "b"}, {Label: "c"}})
	picker.SetCursor(2)

	picker.SetItems([]PickerItem{{Label: "a"}, {Label: "b"}})
	assert.Equal(t, 1, picker.Cursor())

	picker.SetItems(nil)
	assert.Equal(t, 0, picker.Cursor())
	assert.NotPanics(t, func() { picker.HandleKey(pickerKey('j')); _ = picker.View(40, 10) })
}

func TestPicker_ViewScrollsToCursorAndFits(t *testing.T) {
	t.Parallel()

	var items []PickerItem
	for i := 0; i < 30; i++ {
		items = append(items, PickerItem{Label: fmt.Sprintf("item-%02d", i), Detail: "value"})
	}
	picker := NewPicker("Pick", items)
	picker.SetCursor(25)

	view := picker.View(40, 8)

	assert.Contains(t, view, "item-25", "the highlighted item must be on screen")
	assert.NotContains(t, view, "item-00")
	assert.Contains(t, view, "(26/30)")
	assert.LessOrEqual(t, lipgloss.Height(view), 8)
	assert.LessOrEqual(t, lipgloss.Width(view), 40)
}
