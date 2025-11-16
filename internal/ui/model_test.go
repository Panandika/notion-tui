package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/Panandika/notion-tui/internal/config"
)

func TestNewModel(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Debug:       false,
	}

	model := NewModel(cfg)

	assert.Equal(t, "test_token", model.config.NotionToken)
	assert.Equal(t, "test_db_id", model.config.DatabaseID)
	assert.Equal(t, 0, model.cursor)
	assert.Equal(t, 0, len(model.pages))
	assert.False(t, model.ready)
}

func TestModelUpdate_WindowSize(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
	}
	model := NewModel(cfg)

	// Send window size message
	windowMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(windowMsg)
	m := updatedModel.(Model)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
	assert.True(t, m.ready)
}

func TestModelUpdate_Navigation(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
	}
	model := NewModel(cfg)
	model.pages = []string{"Page 1", "Page 2", "Page 3"}
	model.ready = true

	// Test down navigation
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(downMsg)
	m := updatedModel.(Model)
	assert.Equal(t, 1, m.cursor)

	// Test up navigation
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = m.Update(upMsg)
	m = updatedModel.(Model)
	assert.Equal(t, 0, m.cursor)

	// Test vim keybinding (j for down)
	jMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updatedModel, _ = m.Update(jMsg)
	m = updatedModel.(Model)
	assert.Equal(t, 1, m.cursor)

	// Test vim keybinding (k for up)
	kMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updatedModel, _ = m.Update(kMsg)
	m = updatedModel.(Model)
	assert.Equal(t, 0, m.cursor)
}

func TestModelUpdate_Quit(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
	}
	model := NewModel(cfg)

	// Test quit with 'q'
	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(qMsg)

	// Should return a quit command
	assert.NotNil(t, cmd)
}

func TestModelView(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
	}
	model := NewModel(cfg)

	// Before ready
	view := model.View()
	assert.Contains(t, view, "Initializing")

	// After ready
	model.ready = true
	model.pages = []string{"Page 1", "Page 2"}
	view = model.View()
	assert.Contains(t, view, "Notion Pages")
	assert.Contains(t, view, "Page 1")
	assert.Contains(t, view, "Page 2")

	// With error
	model.err = assert.AnError
	view = model.View()
	assert.Contains(t, view, "Error")
}

func TestModelPagesLoaded(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
	}
	model := NewModel(cfg)
	model.pages = []string{"Old Page"}

	msg := pagesLoadedMsg{
		titles: []string{"New Page 1", "New Page 2"},
		ids:    []string{"id1", "id2"},
		err:    nil,
	}

	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	assert.Equal(t, 2, len(m.pages))
	assert.Equal(t, "New Page 1", m.pages[0])
	assert.Equal(t, "New Page 2", m.pages[1])
	assert.Equal(t, "id1", m.pageIDs[0])
	assert.Nil(t, m.err)
}

func TestModelInit(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
	}
	model := NewModel(cfg)

	// Init should return a command
	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestModelCursorBounds(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
	}
	model := NewModel(cfg)
	model.pages = []string{"Page 1", "Page 2", "Page 3"}
	model.ready = true

	// Try to go below 0
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ := model.Update(upMsg)
	m := updatedModel.(Model)
	assert.Equal(t, 0, m.cursor) // Should stay at 0

	// Navigate to last item
	model.cursor = 2
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(downMsg)
	m = updatedModel.(Model)
	assert.Equal(t, 2, m.cursor) // Should stay at last item
}
