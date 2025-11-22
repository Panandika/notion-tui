package components

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
)

// ErrorType represents different categories of errors.
type ErrorType int

const (
	// ErrorTypeUnknown represents an unclassified error.
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeNetwork represents network connectivity errors.
	ErrorTypeNetwork
	// ErrorTypeAuth represents authentication/authorization errors.
	ErrorTypeAuth
	// ErrorTypeNotFound represents resource not found errors.
	ErrorTypeNotFound
	// ErrorTypeRateLimit represents rate limiting errors.
	ErrorTypeRateLimit
	// ErrorTypeServer represents server-side errors.
	ErrorTypeServer
	// ErrorTypeValidation represents validation/bad request errors.
	ErrorTypeValidation
)

// ErrorAction represents an action the user can take in response to an error.
type ErrorAction struct {
	Label string
	Key   string
}

var (
	// ActionRetry is the retry action.
	ActionRetry = ErrorAction{Label: "Retry", Key: "r"}
	// ActionDismiss is the dismiss action.
	ActionDismiss = ErrorAction{Label: "Dismiss", Key: "d"}
	// ActionGoBack is the go back action.
	ActionGoBack = ErrorAction{Label: "Go Back", Key: "esc"}
)

// ErrorViewStyles holds the styles for the error view.
type ErrorViewStyles struct {
	Container lipgloss.Style
	Icon      lipgloss.Style
	Title     lipgloss.Style
	Message   lipgloss.Style
	Context   lipgloss.Style
	Actions   lipgloss.Style
	Action    lipgloss.Style
	Border    lipgloss.Style
}

// DefaultErrorViewStyles returns the default styles for the error view.
func DefaultErrorViewStyles() ErrorViewStyles {
	return ErrorViewStyles{
		Container: lipgloss.NewStyle().
			Padding(2, 4),
		Icon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true).
			MarginTop(1).
			MarginBottom(1),
		Message: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3F4F6")).
			MarginBottom(1),
		Context: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Italic(true).
			MarginBottom(2),
		Actions: lipgloss.NewStyle().
			MarginTop(1),
		Action: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true).
			MarginRight(2),
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#EF4444")).
			Padding(1, 2),
	}
}

// ErrorView displays user-friendly error messages with contextual actions.
type ErrorView struct {
	err        error
	errorType  ErrorType
	message    string
	context    string
	actions    []ErrorAction
	width      int
	height     int
	styles     ErrorViewStyles
	showBorder bool
	retryAfter int // seconds until retry (for rate limit)
}

// NewErrorViewInput contains parameters for creating a new ErrorView.
type NewErrorViewInput struct {
	Err        error
	Width      int
	Height     int
	ShowBorder bool
}

// NewErrorView creates a new ErrorView instance.
func NewErrorView(input NewErrorViewInput) ErrorView {
	ev := ErrorView{
		err:        input.Err,
		width:      input.Width,
		height:     input.Height,
		styles:     DefaultErrorViewStyles(),
		showBorder: input.ShowBorder,
	}

	// Classify error and set appropriate message/actions
	ev.classifyError()

	return ev
}

// classifyError analyzes the error and sets appropriate type, message, and actions.
func (ev *ErrorView) classifyError() {
	if ev.err == nil {
		ev.errorType = ErrorTypeUnknown
		ev.message = "An unknown error occurred"
		ev.actions = []ErrorAction{ActionGoBack}
		return
	}

	errStr := ev.err.Error()

	// Check for HTTP status codes from Notion API
	if httpErr, ok := ev.err.(*notionapi.Error); ok {
		ev.classifyHTTPError(httpErr)
		return
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(ev.err, &netErr) {
		ev.errorType = ErrorTypeNetwork
		ev.message = "Can't connect to Notion"
		ev.context = "Check your internet connection and try again."
		ev.actions = []ErrorAction{ActionRetry, ActionGoBack}
		return
	}

	// Check for common error patterns in error string
	switch {
	case strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized"):
		ev.errorType = ErrorTypeAuth
		ev.message = "Invalid Notion token"
		ev.context = "Please check your NOTION_TOKEN in the config."
		ev.actions = []ErrorAction{ActionGoBack}

	case strings.Contains(errStr, "404") || strings.Contains(errStr, "not found"):
		ev.errorType = ErrorTypeNotFound
		ev.message = "Page not found"
		ev.context = "This page may have been deleted or you don't have access."
		ev.actions = []ErrorAction{ActionGoBack}

	case strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit"):
		ev.errorType = ErrorTypeRateLimit
		ev.message = "Rate limit exceeded"
		ev.context = "Too many requests. Retrying automatically..."
		ev.actions = []ErrorAction{ActionGoBack}

	case strings.Contains(errStr, "500") || strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") || strings.Contains(errStr, "504"):
		ev.errorType = ErrorTypeServer
		ev.message = "Notion server error"
		ev.context = "Notion's servers are having issues. Please try again."
		ev.actions = []ErrorAction{ActionRetry, ActionGoBack}

	case strings.Contains(errStr, "400") || strings.Contains(errStr, "bad request"):
		ev.errorType = ErrorTypeValidation
		ev.message = "Invalid request"
		ev.context = "The request couldn't be processed. Please try again."
		ev.actions = []ErrorAction{ActionGoBack}

	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded"):
		ev.errorType = ErrorTypeNetwork
		ev.message = "Request timed out"
		ev.context = "The request took too long. Check your connection and try again."
		ev.actions = []ErrorAction{ActionRetry, ActionGoBack}

	case strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host"):
		ev.errorType = ErrorTypeNetwork
		ev.message = "Can't reach Notion"
		ev.context = "Check your internet connection and DNS settings."
		ev.actions = []ErrorAction{ActionRetry, ActionGoBack}

	default:
		ev.errorType = ErrorTypeUnknown
		ev.message = "Something went wrong"
		ev.context = fmt.Sprintf("Error: %v", ev.err)
		ev.actions = []ErrorAction{ActionRetry, ActionGoBack}
	}
}

// classifyHTTPError handles notionapi.Error specifically.
func (ev *ErrorView) classifyHTTPError(httpErr *notionapi.Error) {
	switch httpErr.Status {
	case http.StatusUnauthorized:
		ev.errorType = ErrorTypeAuth
		ev.message = "Invalid Notion token"
		ev.context = "Your API token is invalid or expired. Check your config."
		ev.actions = []ErrorAction{ActionGoBack}

	case http.StatusForbidden:
		ev.errorType = ErrorTypeAuth
		ev.message = "Access forbidden"
		ev.context = "You don't have permission to access this resource."
		ev.actions = []ErrorAction{ActionGoBack}

	case http.StatusNotFound:
		ev.errorType = ErrorTypeNotFound
		ev.message = "Page not found"
		ev.context = "This page may have been deleted or moved."
		ev.actions = []ErrorAction{ActionGoBack}

	case http.StatusTooManyRequests:
		ev.errorType = ErrorTypeRateLimit
		ev.message = "Rate limit exceeded"
		if httpErr.Message != "" {
			ev.context = httpErr.Message
		} else {
			ev.context = "Too many requests. Retrying automatically..."
		}
		ev.actions = []ErrorAction{ActionGoBack}

	case http.StatusBadRequest:
		ev.errorType = ErrorTypeValidation
		ev.message = "Invalid request"
		if httpErr.Message != "" {
			ev.context = httpErr.Message
		} else {
			ev.context = "The request couldn't be processed."
		}
		ev.actions = []ErrorAction{ActionGoBack}

	case http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		ev.errorType = ErrorTypeServer
		ev.message = "Notion server error"
		ev.context = "Notion's servers are having issues. Please try again."
		ev.actions = []ErrorAction{ActionRetry, ActionGoBack}

	default:
		ev.errorType = ErrorTypeUnknown
		ev.message = "Something went wrong"
		if httpErr.Message != "" {
			ev.context = httpErr.Message
		} else {
			ev.context = fmt.Sprintf("HTTP %d", httpErr.Status)
		}
		ev.actions = []ErrorAction{ActionRetry, ActionGoBack}
	}
}

// Update handles messages for the error view.
func (ev ErrorView) Update(msg tea.Msg) (ErrorView, tea.Cmd) {
	// ErrorView is currently stateless, but we could add animations or countdown timers
	return ev, nil
}

// View renders the error view.
func (ev ErrorView) View() string {
	// Build error icon based on type
	icon := ev.getErrorIcon()

	// Build title
	title := ev.styles.Title.Render(ev.message)

	// Build context (if any)
	var contextView string
	if ev.context != "" {
		contextView = ev.styles.Context.Render(ev.context)
	}

	// Build actions
	actionStrs := make([]string, 0, len(ev.actions))
	for _, action := range ev.actions {
		actionStr := ev.styles.Action.Render(fmt.Sprintf("%s: %s", action.Key, action.Label))
		actionStrs = append(actionStrs, actionStr)
	}
	actionsView := ev.styles.Actions.Render(strings.Join(actionStrs, " "))

	// Combine all parts
	var content string
	if contextView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left,
			icon,
			title,
			contextView,
			actionsView,
		)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			icon,
			title,
			actionsView,
		)
	}

	// Apply border if requested
	if ev.showBorder {
		content = ev.styles.Border.Render(content)
	}

	// Center in available space
	if ev.width > 0 && ev.height > 0 {
		centered := lipgloss.Place(
			ev.width,
			ev.height,
			lipgloss.Center,
			lipgloss.Center,
			content,
		)
		return centered
	}

	return ev.styles.Container.Render(content)
}

// getErrorIcon returns an appropriate icon for the error type.
func (ev ErrorView) getErrorIcon() string {
	var icon string
	switch ev.errorType {
	case ErrorTypeNetwork:
		icon = "⚠ NETWORK ERROR"
	case ErrorTypeAuth:
		icon = "🔒 AUTHENTICATION ERROR"
	case ErrorTypeNotFound:
		icon = "❓ NOT FOUND"
	case ErrorTypeRateLimit:
		icon = "⏱ RATE LIMIT"
	case ErrorTypeServer:
		icon = "🔧 SERVER ERROR"
	case ErrorTypeValidation:
		icon = "⚠ VALIDATION ERROR"
	default:
		icon = "❌ ERROR"
	}

	return ev.styles.Icon.Render(icon)
}

// SetSize updates the error view dimensions.
func (ev *ErrorView) SetSize(width, height int) {
	ev.width = width
	ev.height = height
}

// SetError updates the error being displayed.
func (ev *ErrorView) SetError(err error) {
	ev.err = err
	ev.classifyError()
}

// Error returns the underlying error.
func (ev ErrorView) Error() error {
	return ev.err
}

// ErrorType returns the classified error type.
func (ev ErrorView) ErrorType() ErrorType {
	return ev.errorType
}

// Message returns the user-friendly error message.
func (ev ErrorView) Message() string {
	return ev.message
}

// Context returns the error context/details.
func (ev ErrorView) Context() string {
	return ev.context
}

// Actions returns the available actions.
func (ev ErrorView) Actions() []ErrorAction {
	return ev.actions
}

// SetStyles updates the error view styles.
func (ev *ErrorView) SetStyles(styles ErrorViewStyles) {
	ev.styles = styles
}

// IsRetryable returns whether this error should allow retry.
func (ev ErrorView) IsRetryable() bool {
	for _, action := range ev.actions {
		if action.Key == ActionRetry.Key {
			return true
		}
	}
	return false
}
