package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandPalette(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		wantWidth        int
		wantHeight       int
		wantIsOpen       bool
		wantCommandCount int
	}{
		{
			name:             "default initialization",
			wantWidth:        60,
			wantHeight:       15,
			wantIsOpen:       false,
			wantCommandCount: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()

			assert.Equal(t, tt.wantWidth, palette.Width())
			assert.Equal(t, tt.wantHeight, palette.Height())
			assert.Equal(t, tt.wantIsOpen, palette.IsOpen())
			assert.Equal(t, tt.wantCommandCount, len(palette.Commands()))
		})
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupFunc  func(p *CommandPalette)
		wantIsOpen bool
	}{
		{
			name:       "open closed palette",
			setupFunc:  func(p *CommandPalette) {},
			wantIsOpen: true,
		},
		{
			name: "open already open palette",
			setupFunc: func(p *CommandPalette) {
				p.Open()
			},
			wantIsOpen: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			tt.setupFunc(&palette)

			palette.Open()

			assert.Equal(t, tt.wantIsOpen, palette.IsOpen())
		})
	}
}

func TestClose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupFunc  func(p *CommandPalette)
		wantIsOpen bool
	}{
		{
			name: "close open palette",
			setupFunc: func(p *CommandPalette) {
				p.Open()
			},
			wantIsOpen: false,
		},
		{
			name:       "close already closed palette",
			setupFunc:  func(p *CommandPalette) {},
			wantIsOpen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			tt.setupFunc(&palette)

			palette.Close()

			assert.Equal(t, tt.wantIsOpen, palette.IsOpen())
		})
	}
}

func TestAddCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		commandName string
		commandDesc string
		wantCount   int
	}{
		{
			name:        "add single custom command",
			commandName: "Custom Command",
			commandDesc: "A custom command description",
			wantCount:   7, // 6 built-in + 1 custom
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			initialCount := len(palette.Commands())

			palette.AddCommand(tt.commandName, tt.commandDesc, "custom", func() tea.Cmd { return nil })

			commands := palette.Commands()
			assert.Equal(t, tt.wantCount, len(commands))
			assert.Equal(t, initialCount+1, len(commands))

			// Verify the added command
			lastCmd := commands[len(commands)-1]
			assert.Equal(t, tt.commandName, lastCmd.Title())
			assert.Equal(t, tt.commandDesc, lastCmd.Description())
			assert.Equal(t, tt.commandName, lastCmd.FilterValue())
		})
	}
}

func TestFuzzySearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		commandNames []string
		wantFilters  []string
	}{
		{
			name:         "filter by command name",
			commandNames: []string{"Search All Pages", "Switch Database", "Refresh Current View", "New Page", "Export Page", "Quit"},
			wantFilters:  []string{"Search All Pages", "Switch Database", "Refresh Current View", "New Page", "Export Page", "Quit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			commands := palette.Commands()

			require.Equal(t, len(tt.commandNames), len(commands))

			for i, cmd := range commands {
				assert.Equal(t, tt.commandNames[i], cmd.Title())
				assert.Equal(t, tt.wantFilters[i], cmd.FilterValue())
			}

			// Verify filter state
			assert.Equal(t, list.Unfiltered, palette.FilterState())
			assert.False(t, palette.IsFiltering())
		})
	}
}

func TestEnterExecute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setupKeys       []string
		wantCommandName string
		wantOpen        bool
	}{
		{
			name:            "execute first command",
			setupKeys:       []string{},
			wantCommandName: "Search All Pages", // First command in new order
			wantOpen:        false,
		},
		{
			name:            "execute second command after navigation",
			setupKeys:       []string{"down"},
			wantCommandName: "Switch Database", // Second command in new order
			wantOpen:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			palette.Open()

			// Apply setup keys
			for _, key := range tt.setupKeys {
				palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			}

			// Execute enter
			palette, cmd := palette.Update(tea.KeyMsg{Type: tea.KeyEnter})

			assert.Equal(t, tt.wantOpen, palette.IsOpen())
			require.NotNil(t, cmd)

			// The command should return a batch containing CommandExecutedMsg
			// We need to execute the batch to get the individual messages
			msgs := []tea.Msg{}
			for _, c := range []tea.Cmd{cmd} {
				if c != nil {
					msg := c()
					if batchMsg, ok := msg.(tea.BatchMsg); ok {
						for _, bc := range batchMsg {
							if bc != nil {
								msgs = append(msgs, bc())
							}
						}
					} else {
						msgs = append(msgs, msg)
					}
				}
			}

			// Find CommandExecutedMsg
			found := false
			for _, msg := range msgs {
				if execMsg, ok := msg.(CommandExecutedMsg); ok {
					assert.Equal(t, tt.wantCommandName, execMsg.CommandName)
					found = true
					break
				}
			}
			assert.True(t, found, "expected CommandExecutedMsg to be returned")
		})
	}
}

func TestEscClose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupFunc  func(p *CommandPalette)
		wantIsOpen bool
	}{
		{
			name: "esc closes open palette",
			setupFunc: func(p *CommandPalette) {
				p.Open()
			},
			wantIsOpen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			tt.setupFunc(&palette)

			palette, cmd := palette.Update(tea.KeyMsg{Type: tea.KeyEscape})

			assert.Equal(t, tt.wantIsOpen, palette.IsOpen())
			assert.Nil(t, cmd)
		})
	}
}

func TestBuiltInCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		wantNames       []string
		wantDescPartial []string
	}{
		{
			name:            "default built-in commands",
			wantNames:       []string{"Search All Pages", "Switch Database", "Refresh Current View", "New Page", "Export Page", "Quit"},
			wantDescPartial: []string{"Search", "Switch", "Refresh", "Create", "Export", "Exit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			commands := palette.Commands()

			require.Equal(t, len(tt.wantNames), len(commands))

			for i, cmd := range commands {
				assert.Equal(t, tt.wantNames[i], cmd.Title())
				assert.Contains(t, cmd.Description(), tt.wantDescPartial[i])
			}
		})
	}
}

func TestCommandPaletteView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		isOpen    bool
		wantEmpty bool
	}{
		{
			name:      "closed palette returns empty view",
			isOpen:    false,
			wantEmpty: true,
		},
		{
			name:      "open palette returns non-empty view",
			isOpen:    true,
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			if tt.isOpen {
				palette.Open()
			}

			view := palette.View()

			if tt.wantEmpty {
				assert.Empty(t, view)
			} else {
				assert.NotEmpty(t, view)
			}
		})
	}
}

func TestCommandItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cmdName    string
		cmdDesc    string
		wantTitle  string
		wantDesc   string
		wantFilter string
	}{
		{
			name:       "basic command",
			cmdName:    "Test Command",
			cmdDesc:    "A test command description",
			wantTitle:  "Test Command",
			wantDesc:   "A test command description",
			wantFilter: "Test Command",
		},
		{
			name:       "command with special characters",
			cmdName:    "Sync & Backup",
			cmdDesc:    "Synchronize and backup data",
			wantTitle:  "Sync & Backup",
			wantDesc:   "Synchronize and backup data",
			wantFilter: "Sync & Backup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := commandItem{
				name:        tt.cmdName,
				description: tt.cmdDesc,
				action:      func() tea.Cmd { return nil },
			}

			assert.Equal(t, tt.wantTitle, cmd.Title())
			assert.Equal(t, tt.wantDesc, cmd.Description())
			assert.Equal(t, tt.wantFilter, cmd.FilterValue())
		})
	}
}

func TestUpdateWhenClosed(t *testing.T) {
	t.Parallel()

	palette := NewCommandPalette()
	assert.False(t, palette.IsOpen())

	// Updates should be no-ops when closed
	updatedPalette, cmd := palette.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.False(t, updatedPalette.IsOpen())
	assert.Nil(t, cmd)
}

func TestInit(t *testing.T) {
	t.Parallel()

	palette := NewCommandPalette()
	cmd := palette.Init()

	assert.Nil(t, cmd)
}

func TestSetSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		width      int
		height     int
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "set larger size",
			width:      80,
			height:     25,
			wantWidth:  80,
			wantHeight: 25,
		},
		{
			name:       "set smaller size",
			width:      40,
			height:     10,
			wantWidth:  40,
			wantHeight: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			palette.SetSize(tt.width, tt.height)

			assert.Equal(t, tt.wantWidth, palette.Width())
			assert.Equal(t, tt.wantHeight, palette.Height())
		})
	}
}

func TestSelectedIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keys      []string
		wantIndex int
	}{
		{
			name:      "initial selection",
			keys:      []string{},
			wantIndex: 0,
		},
		{
			name:      "after navigation down",
			keys:      []string{"down"},
			wantIndex: 1,
		},
		{
			name:      "after navigation up and down",
			keys:      []string{"down", "down", "up"},
			wantIndex: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			palette := NewCommandPalette()
			palette.Open()

			for _, key := range tt.keys {
				palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			}

			assert.Equal(t, tt.wantIndex, palette.SelectedIndex())
		})
	}
}

func TestDefaultCommandPaletteStyles(t *testing.T) {
	t.Parallel()

	styles := DefaultCommandPaletteStyles()

	// Test that styles can render text
	testText := "test"
	assert.NotEmpty(t, styles.Container.Render(testText))
	assert.NotEmpty(t, styles.Title.Render(testText))
}

func TestQuitCommandAction(t *testing.T) {
	t.Parallel()

	palette := NewCommandPalette()
	commands := palette.Commands()

	// Find the Quit command
	var quitCmd commandItem
	found := false
	for _, cmd := range commands {
		if cmd.Title() == "Quit" {
			quitCmd = cmd
			found = true
			break
		}
	}

	require.True(t, found, "Quit command should exist")

	// Test that Quit action returns tea.Quit
	action := quitCmd.action()
	assert.NotNil(t, action)

	msg := action()
	_, isQuitMsg := msg.(tea.QuitMsg)
	assert.True(t, isQuitMsg, "Quit action should return tea.QuitMsg")
}

func TestMultipleAddCommands(t *testing.T) {
	t.Parallel()

	palette := NewCommandPalette()
	initialCount := len(palette.Commands())

	// Add multiple commands
	palette.AddCommand("Custom 1", "First custom", "custom", func() tea.Cmd { return nil })
	palette.AddCommand("Custom 2", "Second custom", "custom", func() tea.Cmd { return nil })
	palette.AddCommand("Custom 3", "Third custom", "custom", func() tea.Cmd { return nil })

	commands := palette.Commands()
	assert.Equal(t, initialCount+3, len(commands))

	// Verify order is preserved
	assert.Equal(t, "Custom 1", commands[initialCount].Title())
	assert.Equal(t, "Custom 2", commands[initialCount+1].Title())
	assert.Equal(t, "Custom 3", commands[initialCount+2].Title())
}
