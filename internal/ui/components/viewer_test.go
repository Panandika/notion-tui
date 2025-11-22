package components

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/testhelpers"
)

func TestNewPageViewer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      NewPageViewerInput
		wantWidth  int
		wantHeight int
		wantReady  bool
	}{
		{
			name: "standard dimensions",
			input: NewPageViewerInput{
				Width:  80,
				Height: 24,
			},
			wantWidth:  80,
			wantHeight: 24,
			wantReady:  false,
		},
		{
			name: "small dimensions",
			input: NewPageViewerInput{
				Width:  40,
				Height: 10,
			},
			wantWidth:  40,
			wantHeight: 10,
			wantReady:  false,
		},
		{
			name: "large dimensions",
			input: NewPageViewerInput{
				Width:  120,
				Height: 50,
			},
			wantWidth:  120,
			wantHeight: 50,
			wantReady:  false,
		},
		{
			name: "zero dimensions",
			input: NewPageViewerInput{
				Width:  0,
				Height: 0,
			},
			wantWidth:  0,
			wantHeight: 0,
			wantReady:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pv := NewPageViewer(tt.input)

			assert.Equal(t, tt.wantWidth, pv.width, "width should match input")
			assert.Equal(t, tt.wantHeight, pv.height, "height should match input")
			assert.Equal(t, tt.wantReady, pv.ready, "should not be ready initially")
			assert.False(t, pv.loading, "should not be loading initially")
			assert.Nil(t, pv.err, "should have no error initially")
			assert.Empty(t, pv.content, "should have no content initially")
			assert.NotNil(t, pv.viewport, "viewport should be initialized")
		})
	}
}

func TestPageViewer_Init(t *testing.T) {
	t.Parallel()

	pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})
	cmd := pv.Init()

	assert.Nil(t, cmd, "Init should return nil command")
}

func TestPageViewer_SetContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		wantReady   bool
		wantLoading bool
	}{
		{
			name:        "simple text",
			content:     "Hello, World!",
			wantReady:   true,
			wantLoading: false,
		},
		{
			name:        "markdown content",
			content:     "# Heading\n\nParagraph text with **bold** and *italic*.",
			wantReady:   true,
			wantLoading: false,
		},
		{
			name:        "empty content",
			content:     "",
			wantReady:   true,
			wantLoading: false,
		},
		{
			name:        "multiline content",
			content:     "Line 1\nLine 2\nLine 3\nLine 4",
			wantReady:   true,
			wantLoading: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})
			pv.SetContent(tt.content)

			assert.Equal(t, tt.content, pv.Content(), "content should match")
			assert.Equal(t, tt.wantReady, pv.IsReady(), "ready state should match")
			assert.Equal(t, tt.wantLoading, pv.IsLoading(), "loading state should match")
			assert.Nil(t, pv.Error(), "should have no error")
		})
	}
}

func TestPageViewer_SetSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initWidth  int
		initHeight int
		newWidth   int
		newHeight  int
	}{
		{
			name:       "increase size",
			initWidth:  80,
			initHeight: 24,
			newWidth:   100,
			newHeight:  30,
		},
		{
			name:       "decrease size",
			initWidth:  100,
			initHeight: 30,
			newWidth:   60,
			newHeight:  20,
		},
		{
			name:       "same size",
			initWidth:  80,
			initHeight: 24,
			newWidth:   80,
			newHeight:  24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pv := NewPageViewer(NewPageViewerInput{
				Width:  tt.initWidth,
				Height: tt.initHeight,
			})

			pv.SetSize(tt.newWidth, tt.newHeight)

			assert.Equal(t, tt.newWidth, pv.width, "width should be updated")
			assert.Equal(t, tt.newHeight, pv.height, "height should be updated")
		})
	}
}

func TestPageViewer_SetBlocks(t *testing.T) {
	t.Parallel()

	// Reset test ID counter for deterministic test results
	testhelpers.ResetTestIDCounter()

	tests := []struct {
		name       string
		blocks     []notionapi.Block
		wantLoaded bool
	}{
		{
			name: "single paragraph",
			blocks: []notionapi.Block{
				testhelpers.NewParagraphBlock("This is a test paragraph."),
			},
			wantLoaded: true,
		},
		{
			name: "multiple block types",
			blocks: []notionapi.Block{
				testhelpers.NewHeading1Block("Main Heading"),
				testhelpers.NewParagraphBlock("Introduction paragraph."),
				testhelpers.NewHeading2Block("Sub Heading"),
				testhelpers.NewBulletedListItemBlock("First item"),
				testhelpers.NewBulletedListItemBlock("Second item"),
			},
			wantLoaded: true,
		},
		{
			name: "code block",
			blocks: []notionapi.Block{
				testhelpers.NewCodeBlock("go", "fmt.Println(\"Hello\")"),
			},
			wantLoaded: true,
		},
		{
			name: "quote and callout",
			blocks: []notionapi.Block{
				testhelpers.NewQuoteBlock("This is a quote."),
				testhelpers.NewCalloutBlock("💡", "This is a callout."),
			},
			wantLoaded: true,
		},
		{
			name:       "empty blocks",
			blocks:     []notionapi.Block{},
			wantLoaded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})

			// Execute the command returned by SetBlocks
			cmd := pv.SetBlocks(tt.blocks)
			require.NotNil(t, cmd, "SetBlocks should return a command")

			// Execute the command to get the message
			msg := cmd()

			// The message should be either ContentLoadedMsg or ErrorMsg
			switch msg := msg.(type) {
			case ContentLoadedMsg:
				assert.Nil(t, msg.Err(), "should not have error")
				if tt.wantLoaded {
					assert.NotEmpty(t, msg.Content(), "should have content")
				}

			case ErrorMsg:
				t.Errorf("unexpected error message: %v", msg)

			default:
				t.Errorf("unexpected message type: %T", msg)
			}
		})
	}
}

func TestPageViewer_Update_ContentLoadedMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		msg            ContentLoadedMsg
		wantReady      bool
		wantLoading    bool
		wantHasError   bool
		wantHasContent bool
	}{
		{
			name:           "successful content load",
			msg:            ContentLoadedMsg{content: "# Test Content\n\nParagraph", err: nil},
			wantReady:      true,
			wantLoading:    false,
			wantHasError:   false,
			wantHasContent: true,
		},
		{
			name:           "content load with error",
			msg:            ContentLoadedMsg{content: "", err: errors.New("conversion failed")},
			wantReady:      false,
			wantLoading:    false,
			wantHasError:   true,
			wantHasContent: false,
		},
		{
			name:           "empty content without error",
			msg:            ContentLoadedMsg{content: "", err: nil},
			wantReady:      true,
			wantLoading:    false,
			wantHasError:   false,
			wantHasContent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})
			pv.loading = true // Simulate loading state

			updated, cmd := pv.Update(tt.msg)

			assert.Nil(t, cmd, "should not return command")
			assert.Equal(t, tt.wantReady, updated.IsReady(), "ready state should match")
			assert.Equal(t, tt.wantLoading, updated.IsLoading(), "loading state should match")

			if tt.wantHasError {
				assert.NotNil(t, updated.Error(), "should have error")
			} else {
				assert.Nil(t, updated.Error(), "should not have error")
			}

			if tt.wantHasContent {
				assert.NotEmpty(t, updated.Content(), "should have content")
			}
		})
	}
}

func TestPageViewer_Update_ErrorMsg(t *testing.T) {
	t.Parallel()

	pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})
	pv.loading = true

	errMsg := ErrorMsg{message: "test error", err: errors.New("underlying error")}

	updated, cmd := pv.Update(errMsg)

	assert.Nil(t, cmd, "should not return command")
	assert.NotNil(t, updated.Error(), "should have error")
	assert.False(t, updated.IsLoading(), "should not be loading")
	assert.False(t, updated.IsReady(), "should not be ready")
}

func TestPageViewer_Update_WindowSizeMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initWidth  int
		initHeight int
		newWidth   int
		newHeight  int
	}{
		{
			name:       "resize larger",
			initWidth:  80,
			initHeight: 24,
			newWidth:   120,
			newHeight:  40,
		},
		{
			name:       "resize smaller",
			initWidth:  100,
			initHeight: 30,
			newWidth:   60,
			newHeight:  20,
		},
		{
			name:       "resize with content",
			initWidth:  80,
			initHeight: 24,
			newWidth:   100,
			newHeight:  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pv := NewPageViewer(NewPageViewerInput{
				Width:  tt.initWidth,
				Height: tt.initHeight,
			})

			// Set some content
			pv.SetContent("Test content")

			msg := tea.WindowSizeMsg{
				Width:  tt.newWidth,
				Height: tt.newHeight,
			}

			updated, cmd := pv.Update(msg)

			assert.Nil(t, cmd, "should not return command")
			assert.Equal(t, tt.newWidth, updated.width, "width should be updated")
			assert.Equal(t, tt.newHeight, updated.height, "height should be updated")
		})
	}
}

func TestPageViewer_Update_KeyMsg(t *testing.T) {
	t.Parallel()

	pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 10})
	pv.SetContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\nLine 11\nLine 12")

	// Test down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, cmd := pv.Update(msg)

	// Viewport should handle the key
	assert.NotNil(t, updated, "should return updated model")
	// cmd might be nil or viewport command
	_ = cmd
}

func TestPageViewer_View(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setup        func(*PageViewer)
		wantContains string
	}{
		{
			name: "loading state",
			setup: func(pv *PageViewer) {
				pv.loading = true
			},
			wantContains: "Loading",
		},
		{
			name: "error state",
			setup: func(pv *PageViewer) {
				pv.err = errors.New("test error")
			},
			wantContains: "Error",
		},
		{
			name: "not ready state",
			setup: func(pv *PageViewer) {
				pv.ready = false
			},
			wantContains: "No content",
		},
		{
			name: "ready with content",
			setup: func(pv *PageViewer) {
				pv.SetContent("Test content is here")
			},
			wantContains: "Test content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})
			tt.setup(&pv)

			view := pv.View()

			assert.NotEmpty(t, view, "view should not be empty")
			assert.Contains(t, view, tt.wantContains, "view should contain expected text")
		})
	}
}

func TestPageViewer_PageID(t *testing.T) {
	t.Parallel()

	pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})

	// Initially empty
	assert.Empty(t, pv.PageID(), "page ID should be empty initially")

	// Set page ID
	pv.SetPageID("test-page-123")
	assert.Equal(t, "test-page-123", pv.PageID(), "page ID should match")
}

func TestPageViewer_ScrollState(t *testing.T) {
	t.Parallel()

	pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 10})

	// Set content that requires scrolling
	longContent := ""
	for i := 0; i < 20; i++ {
		longContent += "Line " + string(rune('0'+i)) + "\n"
	}
	pv.SetContent(longContent)

	// Should be at top initially
	assert.True(t, pv.AtTop(), "should be at top initially")
	assert.False(t, pv.AtBottom(), "should not be at bottom initially")

	// Scroll percent should be available
	percent := pv.ScrollPercent()
	assert.GreaterOrEqual(t, percent, 0.0, "scroll percent should be >= 0")
	assert.LessOrEqual(t, percent, 1.0, "scroll percent should be <= 1")
}

func TestPageViewer_GettersAndSetters(t *testing.T) {
	t.Parallel()

	pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})

	// Test Content()
	assert.Empty(t, pv.Content(), "content should be empty initially")
	pv.SetContent("test content")
	assert.Equal(t, "test content", pv.Content(), "content should match")

	// Test IsReady()
	assert.True(t, pv.IsReady(), "should be ready after SetContent")

	// Test IsLoading()
	assert.False(t, pv.IsLoading(), "should not be loading")

	// Test Error()
	assert.Nil(t, pv.Error(), "should have no error")
}

func TestPageViewer_Integration(t *testing.T) {
	t.Parallel()

	// Reset test ID counter for deterministic test results
	testhelpers.ResetTestIDCounter()

	// Create a viewer
	pv := NewPageViewer(NewPageViewerInput{Width: 80, Height: 24})

	// Set page ID
	pv.SetPageID("integration-test-page")

	// Create test blocks
	blocks := []notionapi.Block{
		testhelpers.NewHeading1Block("Integration Test"),
		testhelpers.NewParagraphBlock("This is an integration test."),
		testhelpers.NewBulletedListItemBlock("Item 1"),
		testhelpers.NewBulletedListItemBlock("Item 2"),
		testhelpers.NewCodeBlock("go", `func main() {
	fmt.Println("Hello")
}`),
	}

	// Load blocks
	cmd := pv.SetBlocks(blocks)
	require.NotNil(t, cmd, "should return command")

	// Execute command
	msg := cmd()

	// Update with the message
	updated, _ := pv.Update(msg)

	// Verify final state
	assert.True(t, updated.IsReady(), "should be ready")
	assert.False(t, updated.IsLoading(), "should not be loading")
	assert.Nil(t, updated.Error(), "should have no error")
	assert.NotEmpty(t, updated.Content(), "should have content")
	assert.Equal(t, "integration-test-page", updated.PageID(), "page ID should match")

	// Verify view renders
	view := updated.View()
	assert.NotEmpty(t, view, "view should not be empty")
}
