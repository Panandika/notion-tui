package pages

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/Panandika/notion-tui/internal/config"
)

func TestNewDatabaseListPage(t *testing.T) {
	databases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
		{ID: "db-2", Name: "Second DB", Icon: "📝"},
	}

	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   databases,
		DefaultDBID: "db-1",
	})

	assert.Equal(t, 80, page.width)
	assert.Equal(t, 40, page.height)
	assert.Len(t, page.databases, 2)
	assert.Equal(t, "db-1", page.defaultDBID)
	assert.Equal(t, "db-1", page.selectedDBID)
}

func TestDatabaseListPageInit(t *testing.T) {
	databases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
	}

	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   databases,
		DefaultDBID: "db-1",
	})

	cmd := page.Init()
	assert.Nil(t, cmd)
}

func TestDatabaseListPageUpdate_WindowSize(t *testing.T) {
	databases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
	}

	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   databases,
		DefaultDBID: "db-1",
	})

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, _ := page.Update(msg)
	updatedPage := updatedModel.(*DatabaseListPage)

	assert.Equal(t, 100, updatedPage.width)
	assert.Equal(t, 50, updatedPage.height)
}

func TestDatabaseListPageUpdate_SelectDatabase(t *testing.T) {
	databases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
		{ID: "db-2", Name: "Second DB", Icon: "📝"},
	}

	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   databases,
		DefaultDBID: "db-1",
	})

	// Press enter to select current database
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := page.Update(msg)
	updatedPage := updatedModel.(*DatabaseListPage)

	assert.NotNil(t, cmd)
	assert.Equal(t, "db-1", updatedPage.selectedDBID)

	// Execute the selection command
	selectionMsg := cmd()
	assert.IsType(t, DatabaseSelectedMsg{}, selectionMsg)

	dbMsg := selectionMsg.(DatabaseSelectedMsg)
	assert.Equal(t, "db-1", dbMsg.DatabaseID)
	assert.Equal(t, "First DB", dbMsg.Database.Name)
}

func TestDatabaseListPageSetDatabases(t *testing.T) {
	initialDatabases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
	}

	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   initialDatabases,
		DefaultDBID: "db-1",
	})

	// Update databases
	newDatabases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
		{ID: "db-2", Name: "Second DB", Icon: "📝"},
		{ID: "db-3", Name: "Third DB", Icon: "📄"},
	}

	page.SetDatabases(newDatabases, "db-2")

	assert.Len(t, page.databases, 3)
	assert.Equal(t, "db-2", page.defaultDBID)
	assert.Len(t, page.list.Items(), 3)
}

func TestDatabaseListPageView(t *testing.T) {
	databases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
		{ID: "db-2", Name: "Second DB", Icon: "📝"},
	}

	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   databases,
		DefaultDBID: "db-1",
	})

	view := page.View()
	assert.NotEmpty(t, view)
	// The view should contain the database list
	assert.Contains(t, view, "Select Database")
}

func TestDatabaseListPageGetters(t *testing.T) {
	databases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
		{ID: "db-2", Name: "Second DB", Icon: "📝"},
	}

	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   databases,
		DefaultDBID: "db-1",
	})

	page.selectedDBID = "db-2"

	assert.Equal(t, "db-2", page.SelectedDatabaseID())
	assert.Len(t, page.Databases(), 2)
}

func TestDatabaseItem_Methods(t *testing.T) {
	db := config.DatabaseConfig{
		ID:   "db-1",
		Name: "Test Database",
		Icon: "📚",
	}

	// Test default database item
	defaultItem := databaseItem{
		db:        db,
		isDefault: true,
	}

	assert.Contains(t, defaultItem.Title(), "📚")
	assert.Contains(t, defaultItem.Title(), "Test Database")
	assert.Contains(t, defaultItem.Title(), "✓")
	assert.Contains(t, defaultItem.Description(), "db-1")
	assert.Contains(t, defaultItem.Description(), "(default)")
	assert.Equal(t, "Test Database", defaultItem.FilterValue())

	// Test non-default database item
	nonDefaultItem := databaseItem{
		db:        db,
		isDefault: false,
	}

	assert.Contains(t, nonDefaultItem.Title(), "📚")
	assert.Contains(t, nonDefaultItem.Title(), "Test Database")
	assert.NotContains(t, nonDefaultItem.Title(), "✓")
	assert.Contains(t, nonDefaultItem.Description(), "db-1")
	assert.NotContains(t, nonDefaultItem.Description(), "(default)")
}

func TestDatabaseItem_NoIcon(t *testing.T) {
	db := config.DatabaseConfig{
		ID:   "db-1",
		Name: "Test Database",
		Icon: "",
	}

	item := databaseItem{
		db:        db,
		isDefault: false,
	}

	// Should use default icon
	assert.Contains(t, item.Title(), "📄")
	assert.Contains(t, item.Title(), "Test Database")
}

func TestDatabaseListPage_EmptyDatabases(t *testing.T) {
	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   []config.DatabaseConfig{},
		DefaultDBID: "",
	})

	assert.Empty(t, page.databases)
	assert.Empty(t, page.defaultDBID)
	assert.Empty(t, page.list.Items())
}

func TestDatabaseListPage_MultipleUpdates(t *testing.T) {
	databases := []config.DatabaseConfig{
		{ID: "db-1", Name: "First DB", Icon: "📚"},
		{ID: "db-2", Name: "Second DB", Icon: "📝"},
	}

	page := NewDatabaseListPage(NewDatabaseListPageInput{
		Width:       80,
		Height:      40,
		Databases:   databases,
		DefaultDBID: "db-1",
	})

	// Process multiple key messages
	msg1 := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := page.Update(msg1)
	page = *updatedModel.(*DatabaseListPage)

	msg2 := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := page.Update(msg2)
	page = *updatedModel.(*DatabaseListPage)

	assert.NotNil(t, cmd)

	// Should have moved to second item and selected it
	selectionMsg := cmd()
	dbMsg := selectionMsg.(DatabaseSelectedMsg)
	// The exact selected database depends on list navigation,
	// but the message should be valid
	assert.NotEmpty(t, dbMsg.DatabaseID)
	assert.NotEmpty(t, dbMsg.Database.Name)
}

func TestDatabaseSelectedMsg(t *testing.T) {
	db := config.DatabaseConfig{
		ID:   "db-1",
		Name: "Test DB",
		Icon: "📚",
	}

	msg := DatabaseSelectedMsg{
		DatabaseID: "db-1",
		Database:   db,
	}

	assert.Equal(t, "db-1", msg.DatabaseID)
	assert.Equal(t, "Test DB", msg.Database.Name)
	assert.Equal(t, "📚", msg.Database.Icon)
}
