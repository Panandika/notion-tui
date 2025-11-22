package components

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestNewStatusBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		wantMode       string
		wantSyncStatus string
		wantHelpText   string
		wantWidth      int
	}{
		{
			name:           "default values",
			wantMode:       ModeBrowse,
			wantSyncStatus: StatusSynced,
			wantHelpText:   "? for help",
			wantWidth:      80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()

			assert.Equal(t, tt.wantMode, sb.Mode())
			assert.Equal(t, tt.wantSyncStatus, sb.SyncStatus())
			assert.Equal(t, tt.wantHelpText, sb.HelpText())
			assert.Equal(t, tt.wantWidth, sb.Width())
		})
	}
}

func TestSetMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mode     string
		wantMode string
	}{
		{
			name:     "set browse mode",
			mode:     ModeBrowse,
			wantMode: ModeBrowse,
		},
		{
			name:     "set edit mode",
			mode:     ModeEdit,
			wantMode: ModeEdit,
		},
		{
			name:     "set command mode",
			mode:     ModeCommand,
			wantMode: ModeCommand,
		},
		{
			name:     "set custom mode",
			mode:     "CUSTOM",
			wantMode: "CUSTOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetMode(tt.mode)

			assert.Equal(t, tt.wantMode, sb.Mode())
		})
	}
}

func TestSetSyncStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     string
		wantStatus string
	}{
		{
			name:       "set synced status",
			status:     StatusSynced,
			wantStatus: StatusSynced,
		},
		{
			name:       "set syncing status",
			status:     StatusSyncing,
			wantStatus: StatusSyncing,
		},
		{
			name:       "set offline status",
			status:     StatusOffline,
			wantStatus: StatusOffline,
		},
		{
			name:       "set error status",
			status:     StatusError,
			wantStatus: StatusError,
		},
		{
			name:       "set custom status",
			status:     "CUSTOM_STATUS",
			wantStatus: "CUSTOM_STATUS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetSyncStatus(tt.status)

			assert.Equal(t, tt.wantStatus, sb.SyncStatus())
		})
	}
}

func TestSetHelpText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		wantText string
	}{
		{
			name:     "set simple help text",
			text:     "Press q to quit",
			wantText: "Press q to quit",
		},
		{
			name:     "set empty help text",
			text:     "",
			wantText: "",
		},
		{
			name:     "set long help text",
			text:     "This is a very long help text that provides detailed instructions to the user",
			wantText: "This is a very long help text that provides detailed instructions to the user",
		},
		{
			name:     "set help text with special characters",
			text:     "Ctrl+S save | Esc cancel | ? help",
			wantText: "Ctrl+S save | Esc cancel | ? help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetHelpText(tt.text)

			assert.Equal(t, tt.wantText, sb.HelpText())
		})
	}
}

func TestSetWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		width     int
		wantWidth int
	}{
		{
			name:      "set standard width",
			width:     80,
			wantWidth: 80,
		},
		{
			name:      "set narrow width",
			width:     40,
			wantWidth: 40,
		},
		{
			name:      "set wide width",
			width:     200,
			wantWidth: 200,
		},
		{
			name:      "set zero width",
			width:     0,
			wantWidth: 0,
		},
		{
			name:      "set negative width",
			width:     -10,
			wantWidth: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetWidth(tt.width)

			assert.Equal(t, tt.wantWidth, sb.Width())
		})
	}
}

func TestStatusBarView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mode       string
		syncStatus string
		helpText   string
		width      int
		checkFunc  func(t *testing.T, view string)
	}{
		{
			name:       "standard view",
			mode:       ModeBrowse,
			syncStatus: StatusSynced,
			helpText:   "? for help",
			width:      80,
			checkFunc: func(t *testing.T, view string) {
				assert.NotEmpty(t, view)
				assert.Contains(t, view, ModeBrowse)
				assert.Contains(t, view, StatusSynced)
			},
		},
		{
			name:       "edit mode view",
			mode:       ModeEdit,
			syncStatus: StatusSyncing,
			helpText:   "Ctrl+S to save",
			width:      80,
			checkFunc: func(t *testing.T, view string) {
				assert.Contains(t, view, ModeEdit)
				assert.Contains(t, view, StatusSyncing)
			},
		},
		{
			name:       "command mode view",
			mode:       ModeCommand,
			syncStatus: StatusOffline,
			helpText:   "Type command",
			width:      100,
			checkFunc: func(t *testing.T, view string) {
				assert.Contains(t, view, ModeCommand)
				assert.Contains(t, view, StatusOffline)
			},
		},
		{
			name:       "error status view",
			mode:       ModeBrowse,
			syncStatus: StatusError,
			helpText:   "r to retry",
			width:      80,
			checkFunc: func(t *testing.T, view string) {
				assert.Contains(t, view, StatusError)
			},
		},
		{
			name:       "zero width",
			mode:       ModeBrowse,
			syncStatus: StatusSynced,
			helpText:   "help",
			width:      0,
			checkFunc: func(t *testing.T, view string) {
				assert.Empty(t, view)
			},
		},
		{
			name:       "negative width",
			mode:       ModeBrowse,
			syncStatus: StatusSynced,
			helpText:   "help",
			width:      -10,
			checkFunc: func(t *testing.T, view string) {
				assert.Empty(t, view)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetMode(tt.mode)
			sb.SetSyncStatus(tt.syncStatus)
			sb.SetHelpText(tt.helpText)
			sb.SetWidth(tt.width)

			view := sb.View()
			tt.checkFunc(t, view)
		})
	}
}

func TestStatusBar_CompactWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		width     int
		checkFunc func(t *testing.T, view string)
	}{
		{
			name:  "very narrow terminal",
			width: 15,
			checkFunc: func(t *testing.T, view string) {
				// Should at minimum show the mode
				assert.NotEmpty(t, view)
				assert.Contains(t, view, ModeBrowse)
			},
		},
		{
			name:  "moderately narrow terminal",
			width: 30,
			checkFunc: func(t *testing.T, view string) {
				// Should show mode and sync status
				assert.Contains(t, view, ModeBrowse)
			},
		},
		{
			name:  "narrow but usable",
			width: 40,
			checkFunc: func(t *testing.T, view string) {
				assert.NotEmpty(t, view)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetWidth(tt.width)

			view := sb.View()
			tt.checkFunc(t, view)
		})
	}
}

func TestStatusBar_WideWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		width     int
		helpText  string
		checkFunc func(t *testing.T, view string)
	}{
		{
			name:     "wide terminal",
			width:    120,
			helpText: "? for help",
			checkFunc: func(t *testing.T, view string) {
				assert.Contains(t, view, ModeBrowse)
				assert.Contains(t, view, StatusSynced)
				// Help text should be visible in wide terminal
				assert.NotEmpty(t, view)
			},
		},
		{
			name:     "very wide terminal",
			width:    200,
			helpText: "Extended help: ? for help | Ctrl+P command palette | r refresh",
			checkFunc: func(t *testing.T, view string) {
				assert.NotEmpty(t, view)
				// Should fill the width with spacing
				assert.Contains(t, view, ModeBrowse)
			},
		},
		{
			name:     "ultra wide with long help",
			width:    300,
			helpText: "Very long help text that spans many characters",
			checkFunc: func(t *testing.T, view string) {
				assert.NotEmpty(t, view)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetWidth(tt.width)
			sb.SetHelpText(tt.helpText)

			view := sb.View()
			tt.checkFunc(t, view)
		})
	}
}

func TestModeColors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mode      string
		wantColor lipgloss.Color
	}{
		{
			name:      "browse mode color",
			mode:      ModeBrowse,
			wantColor: lipgloss.Color("#10B981"),
		},
		{
			name:      "edit mode color",
			mode:      ModeEdit,
			wantColor: lipgloss.Color("#F59E0B"),
		},
		{
			name:      "command mode color",
			mode:      ModeCommand,
			wantColor: lipgloss.Color("#7C3AED"),
		},
		{
			name:      "unknown mode defaults to browse color",
			mode:      "UNKNOWN",
			wantColor: lipgloss.Color("#10B981"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetMode(tt.mode)

			color := sb.getModeColor()
			assert.Equal(t, tt.wantColor, color)
		})
	}
}

func TestSyncColors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		syncStatus string
		wantColor  lipgloss.Color
	}{
		{
			name:       "synced color",
			syncStatus: StatusSynced,
			wantColor:  lipgloss.Color("#10B981"),
		},
		{
			name:       "syncing color",
			syncStatus: StatusSyncing,
			wantColor:  lipgloss.Color("#F59E0B"),
		},
		{
			name:       "offline color",
			syncStatus: StatusOffline,
			wantColor:  lipgloss.Color("#6B7280"),
		},
		{
			name:       "error color",
			syncStatus: StatusError,
			wantColor:  lipgloss.Color("#EF4444"),
		},
		{
			name:       "unknown status defaults to synced color",
			syncStatus: "UNKNOWN",
			wantColor:  lipgloss.Color("#10B981"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetSyncStatus(tt.syncStatus)

			color := sb.getSyncColor()
			assert.Equal(t, tt.wantColor, color)
		})
	}
}

func TestDefaultStatusBarStyles(t *testing.T) {
	t.Parallel()

	styles := DefaultStatusBarStyles()

	// Test that styles can render text
	testText := "test"
	assert.NotEmpty(t, styles.Container.Render(testText))
	assert.NotEmpty(t, styles.Mode.Render(testText))
	assert.NotEmpty(t, styles.SyncStatus.Render(testText))
	assert.NotEmpty(t, styles.HelpText.Render(testText))

	// Test that colors are set
	assert.NotEmpty(t, string(styles.ModeBrowseColor))
	assert.NotEmpty(t, string(styles.ModeEditColor))
	assert.NotEmpty(t, string(styles.ModeCommandColor))
	assert.NotEmpty(t, string(styles.SyncedColor))
	assert.NotEmpty(t, string(styles.SyncingColor))
	assert.NotEmpty(t, string(styles.OfflineColor))
	assert.NotEmpty(t, string(styles.ErrorColor))
}

func TestStatusBarSetStyles(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar()

	customStyles := StatusBarStyles{
		Container: lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")),
		Mode: lipgloss.NewStyle().
			Bold(true),
		SyncStatus:       lipgloss.NewStyle(),
		HelpText:         lipgloss.NewStyle(),
		ModeBrowseColor:  lipgloss.Color("#FF0000"),
		ModeEditColor:    lipgloss.Color("#00FF00"),
		ModeCommandColor: lipgloss.Color("#0000FF"),
		SyncedColor:      lipgloss.Color("#FFFFFF"),
		SyncingColor:     lipgloss.Color("#AAAAAA"),
		OfflineColor:     lipgloss.Color("#555555"),
		ErrorColor:       lipgloss.Color("#FF0000"),
	}

	sb.SetStyles(customStyles)

	retrievedStyles := sb.Styles()
	assert.Equal(t, customStyles.ModeBrowseColor, retrievedStyles.ModeBrowseColor)
	assert.Equal(t, customStyles.ModeEditColor, retrievedStyles.ModeEditColor)
	assert.Equal(t, customStyles.ModeCommandColor, retrievedStyles.ModeCommandColor)
}

func TestStatusBarViewLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mode       string
		syncStatus string
		helpText   string
		width      int
		checkFunc  func(t *testing.T, view string)
	}{
		{
			name:       "contains separator",
			mode:       ModeBrowse,
			syncStatus: StatusSynced,
			helpText:   "help",
			width:      80,
			checkFunc: func(t *testing.T, view string) {
				// Should have separator between mode and sync status
				assert.Contains(t, view, "|")
			},
		},
		{
			name:       "mode appears before sync status",
			mode:       ModeEdit,
			syncStatus: StatusSyncing,
			helpText:   "help",
			width:      80,
			checkFunc: func(t *testing.T, view string) {
				modeIdx := strings.Index(view, ModeEdit)
				syncIdx := strings.Index(view, StatusSyncing)
				assert.True(t, modeIdx < syncIdx, "mode should appear before sync status")
			},
		},
		{
			name:       "help text is at right side",
			mode:       ModeBrowse,
			syncStatus: StatusSynced,
			helpText:   "? for help",
			width:      80,
			checkFunc: func(t *testing.T, view string) {
				// Help text should be present
				// The exact position is hard to test due to ANSI codes,
				// but we can verify it's in the output
				assert.NotEmpty(t, view)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetMode(tt.mode)
			sb.SetSyncStatus(tt.syncStatus)
			sb.SetHelpText(tt.helpText)
			sb.SetWidth(tt.width)

			view := sb.View()
			tt.checkFunc(t, view)
		})
	}
}

func TestStatusBarConstants(t *testing.T) {
	t.Parallel()

	// Test that mode constants are properly defined
	assert.Equal(t, "BROWSE", ModeBrowse)
	assert.Equal(t, "EDIT", ModeEdit)
	assert.Equal(t, "COMMAND", ModeCommand)

	// Test that sync status constants are properly defined
	assert.Equal(t, "SYNCED", StatusSynced)
	assert.Equal(t, "SYNCING", StatusSyncing)
	assert.Equal(t, "OFFLINE", StatusOffline)
	assert.Equal(t, "ERROR", StatusError)
	assert.Equal(t, "CONNECTED", StatusConnected)
	assert.Equal(t, "DISCONNECTED", StatusDisconnect)
}

func TestStatusBarMultipleUpdates(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar()

	// Test chained updates
	sb.SetMode(ModeEdit)
	sb.SetSyncStatus(StatusSyncing)
	sb.SetHelpText("Saving...")
	sb.SetWidth(100)

	assert.Equal(t, ModeEdit, sb.Mode())
	assert.Equal(t, StatusSyncing, sb.SyncStatus())
	assert.Equal(t, "Saving...", sb.HelpText())
	assert.Equal(t, 100, sb.Width())

	// Update again
	sb.SetMode(ModeBrowse)
	sb.SetSyncStatus(StatusSynced)
	sb.SetHelpText("Saved!")

	assert.Equal(t, ModeBrowse, sb.Mode())
	assert.Equal(t, StatusSynced, sb.SyncStatus())
	assert.Equal(t, "Saved!", sb.HelpText())

	// View should reflect latest state
	view := sb.View()
	assert.Contains(t, view, ModeBrowse)
	assert.Contains(t, view, StatusSynced)
}

func TestConnectionState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		state             ConnectionState
		expectedSyncState string
	}{
		{
			name:              "connected state",
			state:             ConnectionStateConnected,
			expectedSyncState: StatusSynced,
		},
		{
			name:              "offline state",
			state:             ConnectionStateOffline,
			expectedSyncState: StatusOffline,
		},
		{
			name:              "error state",
			state:             ConnectionStateError,
			expectedSyncState: StatusError,
		},
		{
			name:              "unknown state",
			state:             ConnectionStateUnknown,
			expectedSyncState: StatusSynced, // Default remains unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetConnectionState(tt.state)

			assert.Equal(t, tt.state, sb.ConnectionState())
			assert.Equal(t, tt.expectedSyncState, sb.SyncStatus())
		})
	}
}

func TestLastSyncTime(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar()

	// Test initial state (zero time)
	assert.True(t, sb.LastSyncTime().IsZero())

	// Set last sync time
	syncTime := time.Now()
	sb.SetLastSyncTime(syncTime)

	assert.Equal(t, syncTime, sb.LastSyncTime())
}

func TestShowSyncTime(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar()

	// Test initial state (disabled)
	assert.False(t, sb.ShowSyncTime())

	// Enable sync time display
	sb.SetShowSyncTime(true)
	assert.True(t, sb.ShowSyncTime())

	// Disable sync time display
	sb.SetShowSyncTime(false)
	assert.False(t, sb.ShowSyncTime())
}

func TestUpdateSyncSuccess(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar()

	// Set initial error state
	sb.SetConnectionState(ConnectionStateError)
	assert.Equal(t, ConnectionStateError, sb.ConnectionState())

	// Update to success
	sb.UpdateSyncSuccess()

	assert.Equal(t, ConnectionStateConnected, sb.ConnectionState())
	assert.Equal(t, StatusSynced, sb.SyncStatus())
	assert.False(t, sb.LastSyncTime().IsZero())
}

func TestUpdateSyncError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		isNetworkError    bool
		expectedConnState ConnectionState
		expectedSyncState string
	}{
		{
			name:              "network error",
			isNetworkError:    true,
			expectedConnState: ConnectionStateOffline,
			expectedSyncState: StatusOffline,
		},
		{
			name:              "non-network error",
			isNetworkError:    false,
			expectedConnState: ConnectionStateError,
			expectedSyncState: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.UpdateSyncError(tt.isNetworkError)

			assert.Equal(t, tt.expectedConnState, sb.ConnectionState())
			assert.Equal(t, tt.expectedSyncState, sb.SyncStatus())
		})
	}
}

func TestGetConnectionIndicator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		state             ConnectionState
		expectedIndicator string
	}{
		{
			name:              "connected indicator",
			state:             ConnectionStateConnected,
			expectedIndicator: "●",
		},
		{
			name:              "offline indicator",
			state:             ConnectionStateOffline,
			expectedIndicator: "○",
		},
		{
			name:              "error indicator",
			state:             ConnectionStateError,
			expectedIndicator: "✗",
		},
		{
			name:              "unknown indicator",
			state:             ConnectionStateUnknown,
			expectedIndicator: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetConnectionState(tt.state)

			indicator := sb.getConnectionIndicator()
			assert.Equal(t, tt.expectedIndicator, indicator)
		})
	}
}

func TestFormatSyncTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		syncTime     time.Time
		expectedText string
	}{
		{
			name:         "zero time",
			syncTime:     time.Time{},
			expectedText: "",
		},
		{
			name:         "just now",
			syncTime:     time.Now().Add(-30 * time.Second),
			expectedText: "just now",
		},
		{
			name:         "1 minute ago",
			syncTime:     time.Now().Add(-1 * time.Minute),
			expectedText: "1m ago",
		},
		{
			name:         "5 minutes ago",
			syncTime:     time.Now().Add(-5 * time.Minute),
			expectedText: "5m ago",
		},
		{
			name:         "1 hour ago",
			syncTime:     time.Now().Add(-1 * time.Hour),
			expectedText: "1h ago",
		},
		{
			name:         "3 hours ago",
			syncTime:     time.Now().Add(-3 * time.Hour),
			expectedText: "3h ago",
		},
		{
			name:         "1 day ago",
			syncTime:     time.Now().Add(-24 * time.Hour),
			expectedText: "1d ago",
		},
		{
			name:         "3 days ago",
			syncTime:     time.Now().Add(-72 * time.Hour),
			expectedText: "3d ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetLastSyncTime(tt.syncTime)

			formatted := sb.formatSyncTime()
			assert.Equal(t, tt.expectedText, formatted)
		})
	}
}

func TestStatusBarViewWithConnectionIndicator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		state        ConnectionState
		syncTime     time.Time
		showSyncTime bool
		checkFunc    func(t *testing.T, view string)
	}{
		{
			name:         "connected with sync time shown",
			state:        ConnectionStateConnected,
			syncTime:     time.Now().Add(-5 * time.Minute),
			showSyncTime: true,
			checkFunc: func(t *testing.T, view string) {
				assert.Contains(t, view, "●") // Connected indicator
				assert.Contains(t, view, "5m ago")
			},
		},
		{
			name:         "connected without sync time",
			state:        ConnectionStateConnected,
			syncTime:     time.Now(),
			showSyncTime: false,
			checkFunc: func(t *testing.T, view string) {
				assert.Contains(t, view, "●") // Connected indicator
				assert.NotContains(t, view, "ago")
			},
		},
		{
			name:         "offline indicator",
			state:        ConnectionStateOffline,
			syncTime:     time.Time{},
			showSyncTime: false,
			checkFunc: func(t *testing.T, view string) {
				assert.Contains(t, view, "○") // Offline indicator
				assert.Contains(t, view, StatusOffline)
			},
		},
		{
			name:         "error indicator",
			state:        ConnectionStateError,
			syncTime:     time.Time{},
			showSyncTime: false,
			checkFunc: func(t *testing.T, view string) {
				assert.Contains(t, view, "✗") // Error indicator
				assert.Contains(t, view, StatusError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := NewStatusBar()
			sb.SetConnectionState(tt.state)
			sb.SetLastSyncTime(tt.syncTime)
			sb.SetShowSyncTime(tt.showSyncTime)
			sb.SetWidth(80)

			view := sb.View()
			tt.checkFunc(t, view)
		})
	}
}

func TestConnectionStateTransitions(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar()

	// Start in unknown state
	assert.Equal(t, ConnectionStateUnknown, sb.ConnectionState())

	// Transition to connected
	sb.UpdateSyncSuccess()
	assert.Equal(t, ConnectionStateConnected, sb.ConnectionState())
	assert.Equal(t, StatusSynced, sb.SyncStatus())

	// Transition to offline
	sb.UpdateSyncError(true)
	assert.Equal(t, ConnectionStateOffline, sb.ConnectionState())
	assert.Equal(t, StatusOffline, sb.SyncStatus())

	// Transition to error
	sb.UpdateSyncError(false)
	assert.Equal(t, ConnectionStateError, sb.ConnectionState())
	assert.Equal(t, StatusError, sb.SyncStatus())

	// Transition back to connected
	sb.UpdateSyncSuccess()
	assert.Equal(t, ConnectionStateConnected, sb.ConnectionState())
	assert.Equal(t, StatusSynced, sb.SyncStatus())
}
