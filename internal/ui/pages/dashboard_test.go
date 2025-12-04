package pages

import (
	"testing"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/ui/components"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardPage_Init(t *testing.T) {
	d := NewDashboardPage(NewDashboardPageInput{
		Width:  100,
		Height: 50,
	})
	if cmd := d.Init(); cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestDashboardPage_Navigation(t *testing.T) {
	cfg := &config.Config{
		Databases: []config.DatabaseConfig{
			{ID: "db1", Name: "Test DB"},
		},
	}
	d := NewDashboardPage(NewDashboardPageInput{
		Width:  100,
		Height: 50,
		Config: cfg,
	})

	// Initial selection should be 0 (Search Workspace)
	if d.selectedIdx != 0 {
		t.Errorf("Expected initial selection 0, got %d", d.selectedIdx)
	}

	// Move down
	d, _ = updateDashboard(d, tea.KeyMsg{Type: tea.KeyDown})
	if d.selectedIdx != 1 {
		t.Errorf("Expected selection 1 after down, got %d", d.selectedIdx)
	}

	// Move down again (to database)
	d, _ = updateDashboard(d, tea.KeyMsg{Type: tea.KeyDown})
	if d.selectedIdx != 2 {
		t.Errorf("Expected selection 2 after down, got %d", d.selectedIdx)
	}

	// Move down (should stay at bottom)
	d, _ = updateDashboard(d, tea.KeyMsg{Type: tea.KeyDown})
	if d.selectedIdx != 2 {
		t.Errorf("Expected selection 2 at bottom, got %d", d.selectedIdx)
	}

	// Move up
	d, _ = updateDashboard(d, tea.KeyMsg{Type: tea.KeyUp})
	if d.selectedIdx != 1 {
		t.Errorf("Expected selection 1 after up, got %d", d.selectedIdx)
	}
}

func TestDashboardPage_Actions(t *testing.T) {
	cfg := &config.Config{
		Databases: []config.DatabaseConfig{
			{ID: "db1", Name: "Test DB"},
		},
	}
	d := NewDashboardPage(NewDashboardPageInput{
		Width:  100,
		Height: 50,
		Config: cfg,
	})

	// Test Search Action
	d.selectedIdx = 0
	_, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Expected command for search action")
	}
	msg := cmd()
	if execMsg, ok := msg.(components.CommandExecutedMsg); !ok || execMsg.ActionType != "search" {
		t.Errorf("Expected search action, got %v", msg)
	}

	// Test Switch DB Action
	d.selectedIdx = 1
	_, cmd = d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Expected command for switch-db action")
	}
	msg = cmd()
	if execMsg, ok := msg.(components.CommandExecutedMsg); !ok || execMsg.ActionType != "switch-db" {
		t.Errorf("Expected switch-db action, got %v", msg)
	}

	// Test Open DB Action
	d.selectedIdx = 2
	_, cmd = d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Expected command for open-db action")
	}
	msg = cmd()
	if dbMsg, ok := msg.(DatabaseSelectedMsg); !ok || dbMsg.DatabaseID != "db1" {
		t.Errorf("Expected DatabaseSelectedMsg with ID db1, got %v", msg)
	}
}

// Helper to cast model back to DashboardPage
func updateDashboard(d DashboardPage, msg tea.Msg) (DashboardPage, tea.Cmd) {
	m, cmd := (&d).Update(msg)
	return *(m.(*DashboardPage)), cmd
}
