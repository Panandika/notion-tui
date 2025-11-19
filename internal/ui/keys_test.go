package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyMap(t *testing.T) {
	t.Parallel()

	keyMap := DefaultKeyMap()

	tests := []struct {
		name         string
		binding      key.Binding
		expectedKeys []string
		expectedHelp string
	}{
		{
			name:         "Up binding",
			binding:      keyMap.Up,
			expectedKeys: []string{"up", "k"},
			expectedHelp: "move up",
		},
		{
			name:         "Down binding",
			binding:      keyMap.Down,
			expectedKeys: []string{"down", "j"},
			expectedHelp: "move down",
		},
		{
			name:         "Left binding",
			binding:      keyMap.Left,
			expectedKeys: []string{"left", "h"},
			expectedHelp: "move left",
		},
		{
			name:         "Right binding",
			binding:      keyMap.Right,
			expectedKeys: []string{"right", "l"},
			expectedHelp: "move right",
		},
		{
			name:         "Enter binding",
			binding:      keyMap.Enter,
			expectedKeys: []string{"enter"},
			expectedHelp: "select",
		},
		{
			name:         "Back binding",
			binding:      keyMap.Back,
			expectedKeys: []string{"esc"},
			expectedHelp: "back",
		},
		{
			name:         "Quit binding",
			binding:      keyMap.Quit,
			expectedKeys: []string{"ctrl+c", "q"},
			expectedHelp: "quit",
		},
		{
			name:         "ShowHelp binding",
			binding:      keyMap.ShowHelp,
			expectedKeys: []string{"?"},
			expectedHelp: "help",
		},
		{
			name:         "Refresh binding",
			binding:      keyMap.Refresh,
			expectedKeys: []string{"r"},
			expectedHelp: "refresh",
		},
		{
			name:         "NewPage binding",
			binding:      keyMap.NewPage,
			expectedKeys: []string{"ctrl+n"},
			expectedHelp: "new page",
		},
		{
			name:         "Edit binding",
			binding:      keyMap.Edit,
			expectedKeys: []string{"e"},
			expectedHelp: "edit",
		},
		{
			name:         "Command binding",
			binding:      keyMap.Command,
			expectedKeys: []string{"ctrl+p"},
			expectedHelp: "command palette",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedKeys, tt.binding.Keys())
			assert.Equal(t, tt.expectedHelp, tt.binding.Help().Desc)
		})
	}
}

func TestKeyMapHelp(t *testing.T) {
	t.Parallel()

	keyMap := DefaultKeyMap()
	bindings := keyMap.Help()

	assert.Len(t, bindings, 12, "Help should return all 12 key bindings")

	expectedBindings := []key.Binding{
		keyMap.Up,
		keyMap.Down,
		keyMap.Left,
		keyMap.Right,
		keyMap.Enter,
		keyMap.Back,
		keyMap.Quit,
		keyMap.ShowHelp,
		keyMap.Refresh,
		keyMap.NewPage,
		keyMap.Edit,
		keyMap.Command,
	}

	for i, expected := range expectedBindings {
		assert.Equal(t, expected, bindings[i])
	}
}

func TestKeyMapShortHelp(t *testing.T) {
	t.Parallel()

	keyMap := DefaultKeyMap()
	bindings := keyMap.ShortHelp()

	assert.Len(t, bindings, 5, "ShortHelp should return 5 essential bindings")

	expectedBindings := []key.Binding{
		keyMap.Up,
		keyMap.Down,
		keyMap.Enter,
		keyMap.Quit,
		keyMap.ShowHelp,
	}

	for i, expected := range expectedBindings {
		assert.Equal(t, expected, bindings[i])
	}
}

func TestKeyMapFullHelp(t *testing.T) {
	t.Parallel()

	keyMap := DefaultKeyMap()
	groups := keyMap.FullHelp()

	assert.Len(t, groups, 4, "FullHelp should return 4 groups of bindings")

	tests := []struct {
		name          string
		groupIndex    int
		expectedCount int
	}{
		{
			name:          "Navigation group",
			groupIndex:    0,
			expectedCount: 4,
		},
		{
			name:          "Action group",
			groupIndex:    1,
			expectedCount: 3,
		},
		{
			name:          "Commands group",
			groupIndex:    2,
			expectedCount: 4,
		},
		{
			name:          "Help group",
			groupIndex:    3,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Len(t, groups[tt.groupIndex], tt.expectedCount)
		})
	}
}

func TestKeyBindingEnabled(t *testing.T) {
	t.Parallel()

	keyMap := DefaultKeyMap()
	bindings := keyMap.Help()

	for _, binding := range bindings {
		assert.True(t, binding.Enabled(), "All default bindings should be enabled")
	}
}
