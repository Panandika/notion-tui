package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Mode constants for the status bar display.
const (
	ModeBrowse  = "BROWSE"
	ModeEdit    = "EDIT"
	ModeCommand = "COMMAND"
)

// Sync status constants for the status bar display.
const (
	StatusSynced     = "SYNCED"
	StatusSyncing    = "SYNCING"
	StatusOffline    = "OFFLINE"
	StatusError      = "ERROR"
	StatusConnected  = "CONNECTED"
	StatusDisconnect = "DISCONNECTED"
)

// StatusBarStyles holds the styles for the status bar.
type StatusBarStyles struct {
	Container  lipgloss.Style
	Mode       lipgloss.Style
	SyncStatus lipgloss.Style
	HelpText   lipgloss.Style

	// Mode-specific colors
	ModeBrowseColor  lipgloss.Color
	ModeEditColor    lipgloss.Color
	ModeCommandColor lipgloss.Color

	// Sync status colors
	SyncedColor  lipgloss.Color
	SyncingColor lipgloss.Color
	OfflineColor lipgloss.Color
	ErrorColor   lipgloss.Color
}

// DefaultStatusBarStyles returns the default styles for the status bar.
func DefaultStatusBarStyles() StatusBarStyles {
	return StatusBarStyles{
		Container: lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(lipgloss.Color("#F3F4F6")),
		Mode: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1),
		SyncStatus: lipgloss.NewStyle().
			Padding(0, 1),
		HelpText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true).
			Padding(0, 1),

		// Mode colors
		ModeBrowseColor:  lipgloss.Color("#10B981"),
		ModeEditColor:    lipgloss.Color("#F59E0B"),
		ModeCommandColor: lipgloss.Color("#7C3AED"),

		// Sync status colors
		SyncedColor:  lipgloss.Color("#10B981"),
		SyncingColor: lipgloss.Color("#F59E0B"),
		OfflineColor: lipgloss.Color("#6B7280"),
		ErrorColor:   lipgloss.Color("#EF4444"),
	}
}

// ConnectionState represents the connection state to Notion API.
type ConnectionState int

const (
	// ConnectionStateUnknown means connection state is not yet determined.
	ConnectionStateUnknown ConnectionState = iota
	// ConnectionStateConnected means successfully connected to Notion API.
	ConnectionStateConnected
	// ConnectionStateOffline means no network connectivity.
	ConnectionStateOffline
	// ConnectionStateError means connection error occurred.
	ConnectionStateError
)

// StatusBar is a component that displays mode, sync status, and help text.
type StatusBar struct {
	mode            string
	syncStatus      string
	helpText        string
	width           int
	styles          StatusBarStyles
	connectionState ConnectionState
	lastSyncTime    time.Time
	showSyncTime    bool
}

// NewStatusBar creates a new status bar with default values.
func NewStatusBar() StatusBar {
	return StatusBar{
		mode:            ModeBrowse,
		syncStatus:      StatusSynced,
		helpText:        "? for help",
		width:           80,
		styles:          DefaultStatusBarStyles(),
		connectionState: ConnectionStateUnknown,
		showSyncTime:    false,
	}
}

// SetMode updates the current mode display.
func (s *StatusBar) SetMode(mode string) {
	s.mode = mode
}

// SetSyncStatus updates the sync status display.
func (s *StatusBar) SetSyncStatus(status string) {
	s.syncStatus = status
}

// SetHelpText updates the help text display.
func (s *StatusBar) SetHelpText(text string) {
	s.helpText = text
}

// SetWidth updates the status bar width.
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// Mode returns the current mode.
func (s StatusBar) Mode() string {
	return s.mode
}

// SyncStatus returns the current sync status.
func (s StatusBar) SyncStatus() string {
	return s.syncStatus
}

// HelpText returns the current help text.
func (s StatusBar) HelpText() string {
	return s.helpText
}

// Width returns the current width.
func (s StatusBar) Width() int {
	return s.width
}

// SetConnectionState updates the connection state.
func (s *StatusBar) SetConnectionState(state ConnectionState) {
	s.connectionState = state

	// Update sync status based on connection state
	switch state {
	case ConnectionStateConnected:
		if s.syncStatus != StatusSyncing {
			s.syncStatus = StatusSynced
		}
	case ConnectionStateOffline:
		s.syncStatus = StatusOffline
	case ConnectionStateError:
		s.syncStatus = StatusError
	}
}

// ConnectionState returns the current connection state.
func (s StatusBar) ConnectionState() ConnectionState {
	return s.connectionState
}

// SetLastSyncTime updates the last successful sync time.
func (s *StatusBar) SetLastSyncTime(t time.Time) {
	s.lastSyncTime = t
}

// LastSyncTime returns the last successful sync time.
func (s StatusBar) LastSyncTime() time.Time {
	return s.lastSyncTime
}

// SetShowSyncTime enables or disables showing the last sync time.
func (s *StatusBar) SetShowSyncTime(show bool) {
	s.showSyncTime = show
}

// ShowSyncTime returns whether sync time is shown.
func (s StatusBar) ShowSyncTime() bool {
	return s.showSyncTime
}

// UpdateSyncSuccess marks a successful sync and updates the connection state.
func (s *StatusBar) UpdateSyncSuccess() {
	s.lastSyncTime = time.Now()
	s.connectionState = ConnectionStateConnected
	s.syncStatus = StatusSynced
}

// UpdateSyncError marks a sync error and updates the connection state.
func (s *StatusBar) UpdateSyncError(isNetworkError bool) {
	if isNetworkError {
		s.connectionState = ConnectionStateOffline
		s.syncStatus = StatusOffline
	} else {
		s.connectionState = ConnectionStateError
		s.syncStatus = StatusError
	}
}

// getModeColor returns the appropriate color for the current mode.
func (s StatusBar) getModeColor() lipgloss.Color {
	switch s.mode {
	case ModeEdit:
		return s.styles.ModeEditColor
	case ModeCommand:
		return s.styles.ModeCommandColor
	default:
		return s.styles.ModeBrowseColor
	}
}

// getSyncColor returns the appropriate color for the current sync status.
func (s StatusBar) getSyncColor() lipgloss.Color {
	switch s.syncStatus {
	case StatusSyncing:
		return s.styles.SyncingColor
	case StatusOffline:
		return s.styles.OfflineColor
	case StatusError:
		return s.styles.ErrorColor
	default:
		return s.styles.SyncedColor
	}
}

// getConnectionIndicator returns a visual indicator for the connection state.
func (s StatusBar) getConnectionIndicator() string {
	switch s.connectionState {
	case ConnectionStateConnected:
		return "●" // Green dot
	case ConnectionStateOffline:
		return "○" // Gray dot
	case ConnectionStateError:
		return "✗" // Red X
	default:
		return "?" // Unknown
	}
}

// formatSyncTime formats the last sync time as a human-readable string.
func (s StatusBar) formatSyncTime() string {
	if s.lastSyncTime.IsZero() {
		return ""
	}

	now := time.Now()
	diff := now.Sub(s.lastSyncTime)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

// View renders the status bar with left + gap + right layout.
func (s StatusBar) View() string {
	if s.width <= 0 {
		return ""
	}

	// Build left side: mode | connection indicator sync status [sync time]
	modeStyle := s.styles.Mode.Foreground(s.getModeColor())
	syncStyle := s.styles.SyncStatus.Foreground(s.getSyncColor())
	indicatorStyle := s.styles.SyncStatus.Foreground(s.getSyncColor())

	modeText := modeStyle.Render(s.mode)
	indicator := indicatorStyle.Render(s.getConnectionIndicator())
	syncText := syncStyle.Render(s.syncStatus)
	separator := s.styles.Container.Render(" | ")

	// Build sync status with optional sync time
	var syncContent string
	if s.showSyncTime && !s.lastSyncTime.IsZero() && s.connectionState == ConnectionStateConnected {
		syncTime := s.formatSyncTime()
		syncContent = fmt.Sprintf("%s %s (%s)", indicator, syncText, syncTime)
	} else {
		syncContent = fmt.Sprintf("%s %s", indicator, syncText)
	}

	leftContent := modeText + separator + s.styles.Container.Render(syncContent)

	// Build right side: help text
	rightContent := s.styles.HelpText.Render(s.helpText)

	// Calculate widths (account for ANSI codes by using lipgloss.Width)
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	totalContentWidth := leftWidth + rightWidth

	// Handle different width scenarios
	if s.width < totalContentWidth {
		// Compact mode: truncate or simplify content
		if s.width < leftWidth+3 {
			// Very narrow: just show mode
			return s.styles.Container.Width(s.width).Render(modeText)
		}
		// Show left side only with padding
		paddingNeeded := s.width - leftWidth
		if paddingNeeded < 0 {
			paddingNeeded = 0
		}
		return s.styles.Container.Render(leftContent + strings.Repeat(" ", paddingNeeded))
	}

	// Normal mode: left + gap + right
	gapWidth := s.width - totalContentWidth
	if gapWidth < 0 {
		gapWidth = 0
	}
	gap := strings.Repeat(" ", gapWidth)

	fullBar := leftContent + gap + rightContent

	return s.styles.Container.Width(s.width).Render(fullBar)
}

// SetStyles updates the status bar styles.
func (s *StatusBar) SetStyles(styles StatusBarStyles) {
	s.styles = styles
}

// Styles returns the current styles.
func (s StatusBar) Styles() StatusBarStyles {
	return s.styles
}
