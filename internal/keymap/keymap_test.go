package keymap

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runes(text string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}
}

func TestDefaults(t *testing.T) {
	t.Parallel()

	m := Default()
	for _, action := range Actions() {
		assert.Equal(t, action.Default, keyPress(action.Default).String(), "%s: the default key press must read back as the default key", action.Name)
		assert.Equal(t, action.Default, m.Key(action.Name))
		assert.NoError(t, checkKey(action.Default), "%s: a default must be a key users may also choose", action.Name)

		got, ok := m.Translate(keyPress(action.Default), action.Scope, false)
		assert.True(t, ok)
		assert.Equal(t, action.Default, got.String(), "defaults pass through untouched")
	}
}

func TestTranslate_ReboundAction(t *testing.T) {
	t.Parallel()

	m, err := New(map[string]string{"edit": "E", "quit": "ctrl+q"})
	require.NoError(t, err)

	got, ok := m.Translate(runes("E"), Page, false)
	assert.True(t, ok)
	assert.Equal(t, "e", got.String(), "the new key does what the default did")

	_, ok = m.Translate(runes("e"), Page, false)
	assert.False(t, ok, "the old key no longer does anything")

	got, ok = m.Translate(runes("e"), Rows, false)
	assert.True(t, ok, "on another screen the old key was never the action's")
	assert.Equal(t, "e", got.String())

	got, ok = m.Translate(tea.KeyMsg{Type: tea.KeyCtrlQ}, Rows, false)
	assert.True(t, ok)
	assert.Equal(t, "q", got.String(), "global actions work on every screen")
	_, ok = m.Translate(runes("q"), Tree, false)
	assert.False(t, ok)
}

func TestTranslate_SwappedKeys(t *testing.T) {
	t.Parallel()

	m, err := New(map[string]string{"edit": "p", "properties": "e"})
	require.NoError(t, err)

	got, ok := m.Translate(runes("p"), Page, false)
	assert.True(t, ok)
	assert.Equal(t, "e", got.String())
	got, ok = m.Translate(runes("e"), Page, false)
	assert.True(t, ok)
	assert.Equal(t, "p", got.String())
}

func TestTranslate_WhileTyping(t *testing.T) {
	t.Parallel()

	m, err := New(map[string]string{"edit": "E", "quit": "x", "palette": "ctrl+k"})
	require.NoError(t, err)

	for _, text := range []string{"E", "e", "x", "q"} {
		got, ok := m.Translate(runes(text), Page, true)
		assert.True(t, ok, "%q is text while typing", text)
		assert.Equal(t, text, got.String())
	}
	got, ok := m.Translate(tea.KeyMsg{Type: tea.KeySpace}, Page, true)
	assert.True(t, ok)
	assert.Equal(t, " ", got.String())

	got, ok = m.Translate(tea.KeyMsg{Type: tea.KeyCtrlK}, Page, true)
	assert.True(t, ok)
	assert.Equal(t, "ctrl+p", got.String(), "a global action on a control key still works")
	_, ok = m.Translate(tea.KeyMsg{Type: tea.KeyCtrlP}, Page, true)
	assert.False(t, ok)
}

func TestTranslate_PasteIsNeverAKey(t *testing.T) {
	t.Parallel()

	m, err := New(map[string]string{"edit": "E"})
	require.NoError(t, err)

	paste := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e"), Paste: true}
	got, ok := m.Translate(paste, Page, false)
	assert.True(t, ok)
	assert.Equal(t, paste, got)
}

func TestNew_Rejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		overrides map[string]string
		wantErr   string
	}{
		{"unknown action", map[string]string{"explode": "x"}, `unknown action "explode"`},
		{"empty key", map[string]string{"edit": ""}, "no key given"},
		{"movement key", map[string]string{"edit": "j"}, "cannot be reassigned"},
		{"quit key", map[string]string{"edit": "ctrl+c"}, "cannot be reassigned"},
		{"not a key", map[string]string{"edit": "super+e"}, "is not a key"},
		{"terminal sends tab", map[string]string{"edit": "ctrl+i"}, "is not a key"},
		{"same screen", map[string]string{"edit": "p"}, `"p" is bound to both edit and properties`},
		{"global against a screen", map[string]string{"quit": "s"}, `"s" is bound to both quit and row_sort`},
		{"two overrides", map[string]string{"edit": "x", "todos": "x"}, `"x" is bound to both edit and todos`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tt.overrides)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNew_SameKeyOnDifferentScreensIsFine(t *testing.T) {
	t.Parallel()

	m, err := New(map[string]string{"reload": "R", "row_reload": "R", "tree_reload": "R"})
	require.NoError(t, err)

	for _, screen := range []Scope{Page, Rows, Tree} {
		got, ok := m.Translate(runes("R"), screen, false)
		assert.True(t, ok)
		assert.Equal(t, "r", got.String())
	}
}

func TestLabel(t *testing.T) {
	t.Parallel()

	m, err := New(map[string]string{"find_next": "ctrl+g"})
	require.NoError(t, err)

	assert.Equal(t, "/  ctrl+g  N", m.Label("{find}  {find_next}  {find_previous}"))
	assert.Equal(t, "{nothing} / ctrl+c", m.Label("{nothing} / ctrl+c"))
	assert.Equal(t, "q", (*Map)(nil).Label("{quit}"), "a missing map means defaults")
}
