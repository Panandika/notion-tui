package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModal(t *testing.T) {
	t.Parallel()

	actions := []ModalAction{
		{Label: "Yes", Key: "y", Value: "yes"},
		{Label: "No", Key: "n", Value: "no"},
	}

	input := NewModalInput{
		Title:   "Confirm",
		Message: "Are you sure?",
		Actions: actions,
		Width:   80,
		Height:  24,
	}

	modal := NewModal(input)

	if modal.title != "Confirm" {
		t.Errorf("expected title 'Confirm', got %s", modal.title)
	}
	if modal.message != "Are you sure?" {
		t.Errorf("expected message 'Are you sure?', got %s", modal.message)
	}
	if len(modal.actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(modal.actions))
	}
	if modal.width != 80 {
		t.Errorf("expected width 80, got %d", modal.width)
	}
	if modal.height != 24 {
		t.Errorf("expected height 24, got %d", modal.height)
	}
}

func TestModalUpdate_ActionKey(t *testing.T) {
	t.Parallel()

	actions := []ModalAction{
		{Label: "Save", Key: "s", Value: "save"},
		{Label: "Discard", Key: "d", Value: "discard"},
		{Label: "Cancel", Key: "c", Value: "cancel"},
	}

	modal := NewModal(NewModalInput{
		Title:   "Unsaved Changes",
		Message: "You have unsaved changes. What do you want to do?",
		Actions: actions,
		Width:   80,
		Height:  24,
	})

	tests := []struct {
		name          string
		key           string
		expectedValue string
	}{
		{"save action", "s", "save"},
		{"discard action", "d", "discard"},
		{"cancel action", "c", "cancel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(tt.key[0])}}
			_, cmd := modal.Update(msg)

			if cmd == nil {
				t.Fatal("expected command, got nil")
			}

			resultMsg := cmd()
			responseMsg, ok := resultMsg.(ModalResponseMsg)
			if !ok {
				t.Fatalf("expected ModalResponseMsg, got %T", resultMsg)
			}

			if responseMsg.Value != tt.expectedValue {
				t.Errorf("expected value %s, got %s", tt.expectedValue, responseMsg.Value)
			}
		})
	}
}

func TestModalUpdate_Escape(t *testing.T) {
	t.Parallel()

	actions := []ModalAction{
		{Label: "Yes", Key: "y", Value: "yes"},
		{Label: "No", Key: "n", Value: "no"},
	}

	modal := NewModal(NewModalInput{
		Title:   "Confirm",
		Message: "Are you sure?",
		Actions: actions,
		Width:   80,
		Height:  24,
	})

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := modal.Update(msg)

	if cmd == nil {
		t.Fatal("expected command, got nil")
	}

	resultMsg := cmd()
	_, ok := resultMsg.(ModalDismissMsg)
	if !ok {
		t.Fatalf("expected ModalDismissMsg, got %T", resultMsg)
	}
}

func TestModalUpdate_UnknownKey(t *testing.T) {
	t.Parallel()

	actions := []ModalAction{
		{Label: "Yes", Key: "y", Value: "yes"},
	}

	modal := NewModal(NewModalInput{
		Title:   "Confirm",
		Message: "Are you sure?",
		Actions: actions,
		Width:   80,
		Height:  24,
	})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	_, cmd := modal.Update(msg)

	if cmd != nil {
		t.Error("expected nil command for unknown key")
	}
}

func TestModalView(t *testing.T) {
	t.Parallel()

	actions := []ModalAction{
		{Label: "Yes", Key: "y", Value: "yes"},
		{Label: "No", Key: "n", Value: "no"},
	}

	modal := NewModal(NewModalInput{
		Title:   "Confirm",
		Message: "Are you sure?",
		Actions: actions,
		Width:   80,
		Height:  24,
	})

	view := modal.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestModalSetSize(t *testing.T) {
	t.Parallel()

	modal := NewModal(NewModalInput{
		Title:   "Test",
		Message: "Test message",
		Actions: []ModalAction{{Label: "OK", Key: "o", Value: "ok"}},
		Width:   80,
		Height:  24,
	})

	modal.SetSize(100, 30)

	if modal.width != 100 {
		t.Errorf("expected width 100, got %d", modal.width)
	}
	if modal.height != 30 {
		t.Errorf("expected height 30, got %d", modal.height)
	}
}

func TestModalSetTitle(t *testing.T) {
	t.Parallel()

	modal := NewModal(NewModalInput{
		Title:   "Original",
		Message: "Test",
		Actions: []ModalAction{{Label: "OK", Key: "o", Value: "ok"}},
	})

	modal.SetTitle("New Title")

	if modal.title != "New Title" {
		t.Errorf("expected title 'New Title', got %s", modal.title)
	}
}

func TestModalSetMessage(t *testing.T) {
	t.Parallel()

	modal := NewModal(NewModalInput{
		Title:   "Test",
		Message: "Original message",
		Actions: []ModalAction{{Label: "OK", Key: "o", Value: "ok"}},
	})

	modal.SetMessage("New message")

	if modal.message != "New message" {
		t.Errorf("expected message 'New message', got %s", modal.message)
	}
}

func TestModalSetActions(t *testing.T) {
	t.Parallel()

	originalActions := []ModalAction{
		{Label: "Yes", Key: "y", Value: "yes"},
	}

	modal := NewModal(NewModalInput{
		Title:   "Test",
		Message: "Test",
		Actions: originalActions,
	})

	newActions := []ModalAction{
		{Label: "Save", Key: "s", Value: "save"},
		{Label: "Cancel", Key: "c", Value: "cancel"},
	}

	modal.SetActions(newActions)

	if len(modal.actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(modal.actions))
	}
	if modal.actions[0].Label != "Save" {
		t.Errorf("expected first action 'Save', got %s", modal.actions[0].Label)
	}
}

func TestModalGetters(t *testing.T) {
	t.Parallel()

	actions := []ModalAction{
		{Label: "Yes", Key: "y", Value: "yes"},
	}

	modal := NewModal(NewModalInput{
		Title:   "Test Title",
		Message: "Test Message",
		Actions: actions,
	})

	if modal.Title() != "Test Title" {
		t.Errorf("expected Title() 'Test Title', got %s", modal.Title())
	}
	if modal.Message() != "Test Message" {
		t.Errorf("expected Message() 'Test Message', got %s", modal.Message())
	}
	if len(modal.Actions()) != 1 {
		t.Errorf("expected Actions() length 1, got %d", len(modal.Actions()))
	}
}
