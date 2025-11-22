package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ModalAction represents an action button in the modal.
type ModalAction struct {
	Label string
	Key   string
	Value string // Value returned when this action is selected
}

// ModalResponseMsg is sent when the user selects a modal action.
type ModalResponseMsg struct {
	Value string
}

// ModalDismissMsg is sent when the modal is dismissed without selection.
type ModalDismissMsg struct{}

// ModalStyles holds the styles for the modal.
type ModalStyles struct {
	Overlay   lipgloss.Style
	Container lipgloss.Style
	Title     lipgloss.Style
	Message   lipgloss.Style
	Actions   lipgloss.Style
	Action    lipgloss.Style
	Border    lipgloss.Style
}

// DefaultModalStyles returns the default styles for the modal.
func DefaultModalStyles() ModalStyles {
	return ModalStyles{
		Overlay: lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#FFFFFF")),
		Container: lipgloss.NewStyle().
			Padding(2, 4),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true).
			MarginBottom(1),
		Message: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3F4F6")).
			MarginBottom(2),
		Actions: lipgloss.NewStyle().
			MarginTop(1),
		Action: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true).
			MarginRight(2),
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F59E0B")).
			Padding(1, 2),
	}
}

// Modal displays a confirmation dialog with customizable actions.
type Modal struct {
	title   string
	message string
	actions []ModalAction
	width   int
	height  int
	styles  ModalStyles
}

// NewModalInput contains parameters for creating a new Modal.
type NewModalInput struct {
	Title   string
	Message string
	Actions []ModalAction
	Width   int
	Height  int
}

// NewModal creates a new Modal instance.
func NewModal(input NewModalInput) Modal {
	return Modal{
		title:   input.Title,
		message: input.Message,
		actions: input.Actions,
		width:   input.Width,
		height:  input.Height,
		styles:  DefaultModalStyles(),
	}
}

// Update handles messages for the modal.
func (m Modal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Check if any action key matches
		for _, action := range m.actions {
			if msg.String() == action.Key {
				return m, func() tea.Msg {
					return ModalResponseMsg{Value: action.Value}
				}
			}
		}

		// Handle escape to dismiss
		if msg.String() == "esc" {
			return m, func() tea.Msg {
				return ModalDismissMsg{}
			}
		}
	}

	return m, nil
}

// View renders the modal.
func (m Modal) View() string {
	// Build title
	titleView := m.styles.Title.Render(m.title)

	// Build message
	messageView := m.styles.Message.Render(m.message)

	// Build actions
	actionStrs := make([]string, 0, len(m.actions))
	for _, action := range m.actions {
		actionStr := m.styles.Action.Render(action.Key + ": " + action.Label)
		actionStrs = append(actionStrs, actionStr)
	}
	actionsView := m.styles.Actions.Render(lipgloss.JoinHorizontal(lipgloss.Left, actionStrs...))

	// Combine all parts
	content := lipgloss.JoinVertical(lipgloss.Left,
		titleView,
		messageView,
		actionsView,
	)

	// Apply border
	content = m.styles.Border.Render(content)

	// Center in available space with semi-transparent overlay effect
	if m.width > 0 && m.height > 0 {
		centered := lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			content,
		)
		return centered
	}

	return m.styles.Container.Render(content)
}

// SetSize updates the modal dimensions.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetTitle updates the modal title.
func (m *Modal) SetTitle(title string) {
	m.title = title
}

// SetMessage updates the modal message.
func (m *Modal) SetMessage(message string) {
	m.message = message
}

// SetActions updates the modal actions.
func (m *Modal) SetActions(actions []ModalAction) {
	m.actions = actions
}

// Title returns the modal title.
func (m Modal) Title() string {
	return m.title
}

// Message returns the modal message.
func (m Modal) Message() string {
	return m.message
}

// Actions returns the modal actions.
func (m Modal) Actions() []ModalAction {
	return m.actions
}

// SetStyles updates the modal styles.
func (m *Modal) SetStyles(styles ModalStyles) {
	m.styles = styles
}

// Styles returns the current styles.
func (m Modal) Styles() ModalStyles {
	return m.styles
}
