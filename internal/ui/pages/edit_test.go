package pages

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/testhelpers"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

func TestNewEditPage(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()

	input := NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	}

	ep := NewEditPage(input)

	if ep.pageID != "page-123" {
		t.Errorf("expected pageID page-123, got %s", ep.pageID)
	}
	if ep.blockID != "block-456" {
		t.Errorf("expected blockID block-456, got %s", ep.blockID)
	}
	if !ep.loading {
		t.Error("expected loading to be true initially")
	}
	if ep.width != 80 {
		t.Errorf("expected width 80, got %d", ep.width)
	}
	if ep.height != 24 {
		t.Errorf("expected height 24, got %d", ep.height)
	}
}

func TestEditPageInit(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")
	mockClient.WithBlock(block)

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	cmd := ep.Init()
	if cmd == nil {
		t.Fatal("expected Init to return a command")
	}

	// Execute the command
	msg := cmd()
	loadedMsg, ok := msg.(blockLoadedMsg)
	if !ok {
		t.Fatalf("expected blockLoadedMsg, got %T", msg)
	}

	if loadedMsg.err != nil {
		t.Errorf("expected no error, got %v", loadedMsg.err)
	}
	if loadedMsg.text != "Test content" {
		t.Errorf("expected text 'Test content', got %s", loadedMsg.text)
	}

	// Verify GetBlock was called
	if mockClient.GetBlockCallCount() != 1 {
		t.Errorf("expected 1 GetBlock call, got %d", mockClient.GetBlockCallCount())
	}
}

func TestEditPageLoadBlock_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		block        notionapi.Block
		expectedText string
	}{
		{
			name:         "paragraph block",
			block:        testhelpers.NewParagraphBlock("Paragraph text"),
			expectedText: "Paragraph text",
		},
		{
			name:         "heading1 block",
			block:        testhelpers.NewHeading1Block("Heading 1"),
			expectedText: "Heading 1",
		},
		{
			name:         "heading2 block",
			block:        testhelpers.NewHeading2Block("Heading 2"),
			expectedText: "Heading 2",
		},
		{
			name:         "heading3 block",
			block:        testhelpers.NewHeading3Block("Heading 3"),
			expectedText: "Heading 3",
		},
		{
			name:         "bulleted list item",
			block:        testhelpers.NewBulletedListItemBlock("List item"),
			expectedText: "List item",
		},
		{
			name:         "numbered list item",
			block:        testhelpers.NewNumberedListItemBlock("Numbered item"),
			expectedText: "Numbered item",
		},
		{
			name:         "todo block",
			block:        testhelpers.NewToDoBlock("Todo item", false),
			expectedText: "Todo item",
		},
		{
			name:         "code block",
			block:        testhelpers.NewCodeBlock("go", "fmt.Println(\"Hello\")"),
			expectedText: "fmt.Println(\"Hello\")",
		},
		{
			name:         "quote block",
			block:        testhelpers.NewQuoteBlock("Quoted text"),
			expectedText: "Quoted text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := testhelpers.NewMockNotionClient()
			mockClient.WithBlock(tt.block)

			ep := NewEditPage(NewEditPageInput{
				Width:        80,
				Height:       24,
				NotionClient: mockClient,
				PageID:       "page-123",
				BlockID:      "block-456",
			})

			cmd := ep.Init()
			msg := cmd()
			loadedMsg, ok := msg.(blockLoadedMsg)
			if !ok {
				t.Fatalf("expected blockLoadedMsg, got %T", msg)
			}

			if loadedMsg.err != nil {
				t.Errorf("expected no error, got %v", loadedMsg.err)
			}
			if loadedMsg.text != tt.expectedText {
				t.Errorf("expected text %q, got %q", tt.expectedText, loadedMsg.text)
			}
		})
	}
}

func TestEditPageLoadBlock_Error(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	mockClient.WithError(testhelpers.ErrNotFound)

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	cmd := ep.Init()
	msg := cmd()
	loadedMsg, ok := msg.(blockLoadedMsg)
	if !ok {
		t.Fatalf("expected blockLoadedMsg, got %T", msg)
	}

	if loadedMsg.err == nil {
		t.Error("expected error, got nil")
	}
}

func TestEditPageUpdate_BlockLoaded(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	msg := blockLoadedMsg{
		block: block,
		text:  "Test content",
	}

	model, cmd := ep.Update(msg)
	epPtr, ok := model.(*EditPage)
	if !ok {
		t.Fatalf("expected *EditPage, got %T", model)
	}
	ep = *epPtr

	if ep.loading {
		t.Error("expected loading to be false after blockLoadedMsg")
	}
	if ep.blockType != string(notionapi.BlockTypeParagraph) {
		t.Errorf("expected blockType paragraph, got %s", ep.blockType)
	}
	if ep.originalText != "Test content" {
		t.Errorf("expected originalText 'Test content', got %s", ep.originalText)
	}
	if cmd == nil {
		t.Error("expected cmd to initialize editor")
	}
}

func TestEditPageUpdate_BlockLoadedError(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	msg := blockLoadedMsg{
		err: fmt.Errorf("load error"),
	}

	model, _ := ep.Update(msg)
	epPtr, ok := model.(*EditPage)
	if !ok {
		t.Fatalf("expected *EditPage, got %T", model)
	}
	ep = *epPtr

	if ep.loading {
		t.Error("expected loading to be false after error")
	}
	if ep.err == nil {
		t.Error("expected error to be set")
	}
}

func TestEditPageUpdate_SaveDraft(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Original text")
	mockClient.WithBlock(block)

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// First load the block
	loadedMsg := blockLoadedMsg{
		block: block,
		text:  "Original text",
	}
	model, _ := ep.Update(loadedMsg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	// Now simulate save
	saveMsg := components.SaveDraftMsg{
		BlockID:   "block-456",
		BlockType: string(notionapi.BlockTypeParagraph),
		Content:   "Updated text",
	}

	model, cmd := ep.Update(saveMsg)
	epPtr = model.(*EditPage)
	ep = *epPtr

	if !ep.saving {
		t.Error("expected saving to be true")
	}
	if cmd == nil {
		t.Fatal("expected save command")
	}

	// Execute save command
	resultMsg := cmd()
	savedMsg, ok := resultMsg.(blockSavedMsg)
	if !ok {
		t.Fatalf("expected blockSavedMsg, got %T", resultMsg)
	}

	if savedMsg.err != nil {
		t.Errorf("expected no error, got %v", savedMsg.err)
	}
	if !savedMsg.success {
		t.Error("expected success to be true")
	}

	// Verify UpdateBlock was called
	if mockClient.UpdateBlockCallCount() != 1 {
		t.Errorf("expected 1 UpdateBlock call, got %d", mockClient.UpdateBlockCallCount())
	}
}

func TestEditPageUpdate_BlockSaved_Success(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// Load block first
	loadedMsg := blockLoadedMsg{
		block: block,
		text:  "Test content",
	}
	model, _ := ep.Update(loadedMsg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	// Set saving state
	ep.saving = true

	// Simulate successful save
	msg := blockSavedMsg{success: true}
	model, cmd := ep.Update(msg)
	epPtr = model.(*EditPage)
	ep = *epPtr

	if ep.saving {
		t.Error("expected saving to be false after success")
	}
	if !ep.saved {
		t.Error("expected saved to be true")
	}
	if cmd == nil {
		t.Error("expected cmd to clear saved message")
	}
}

func TestEditPageUpdate_BlockSaved_Error(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// Load block first
	loadedMsg := blockLoadedMsg{
		block: block,
		text:  "Test content",
	}
	model, _ := ep.Update(loadedMsg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	// Set saving state
	ep.saving = true

	// Simulate save error
	msg := blockSavedMsg{err: fmt.Errorf("save error")}
	model, _ = ep.Update(msg)
	epPtr = model.(*EditPage)
	ep = *epPtr

	if ep.saving {
		t.Error("expected saving to be false after error")
	}
	if ep.err == nil {
		t.Error("expected error to be set")
	}
}

func TestEditPageUpdate_CancelEdit(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// Load block first
	loadedMsg := blockLoadedMsg{
		block: block,
		text:  "Test content",
	}
	model, _ := ep.Update(loadedMsg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	// Simulate cancel
	msg := components.CancelEditMsg{BlockID: "block-456"}
	model, cmd := ep.Update(msg)

	// Should return quit command
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestEditPageUpdate_WindowSize(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	model, _ := ep.Update(msg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	if ep.width != 100 {
		t.Errorf("expected width 100, got %d", ep.width)
	}
	if ep.height != 30 {
		t.Errorf("expected height 30, got %d", ep.height)
	}
}

func TestEditPageUpdate_CtrlS(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")
	mockClient.WithBlock(block)

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// Load block first
	loadedMsg := blockLoadedMsg{
		block: block,
		text:  "Test content",
	}
	model, _ := ep.Update(loadedMsg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	// Simulate Ctrl+S
	msg := tea.KeyMsg{Type: tea.KeyCtrlS}
	model, cmd := ep.Update(msg)
	epPtr = model.(*EditPage)
	ep = *epPtr

	if !ep.saving {
		t.Error("expected saving to be true after Ctrl+S")
	}
	if cmd == nil {
		t.Error("expected save command")
	}
}

func TestEditPageUpdate_Escape(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")
	mockClient.WithBlock(block)

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// Load block first
	loadedMsg := blockLoadedMsg{
		block: block,
		text:  "Test content",
	}
	model, _ := ep.Update(loadedMsg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	// Simulate Escape
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, cmd := ep.Update(msg)

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestEditPageView_Loading(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	view := ep.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	// View should contain loading message
	// Note: We can't easily check rendered content due to lipgloss styling
}

func TestEditPageView_Error(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	ep.loading = false
	ep.err = fmt.Errorf("test error")

	view := ep.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestEditPageView_Loaded(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// Load block
	loadedMsg := blockLoadedMsg{
		block: block,
		text:  "Test content",
	}
	model, _ := ep.Update(loadedMsg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	view := ep.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestEditPageSave(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Test content")
	mockClient.WithBlock(block)

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// Load block first
	loadedMsg := blockLoadedMsg{
		block: block,
		text:  "Test content",
	}
	model, _ := ep.Update(loadedMsg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	// Call Save
	cmd := ep.Save()
	if cmd == nil {
		t.Fatal("expected save command")
	}

	// Execute command
	msg := cmd()
	savedMsg, ok := msg.(blockSavedMsg)
	if !ok {
		t.Fatalf("expected blockSavedMsg, got %T", msg)
	}

	if savedMsg.err != nil {
		t.Errorf("expected no error, got %v", savedMsg.err)
	}
	if !savedMsg.success {
		t.Error("expected success to be true")
	}
}

func TestEditPageSave_AlreadySaving(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	ep.loading = false
	ep.saving = true

	cmd := ep.Save()
	if cmd != nil {
		t.Error("expected nil command when already saving")
	}
}

func TestExtractBlockText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    notionapi.Block
		expected string
	}{
		{
			name:     "paragraph",
			block:    testhelpers.NewParagraphBlock("Paragraph text"),
			expected: "Paragraph text",
		},
		{
			name:     "heading1",
			block:    testhelpers.NewHeading1Block("H1 text"),
			expected: "H1 text",
		},
		{
			name:     "code",
			block:    testhelpers.NewCodeBlock("go", "package main"),
			expected: "package main",
		},
		{
			name:     "quote",
			block:    testhelpers.NewQuoteBlock("Quote text"),
			expected: "Quote text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBlockText(tt.block)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBuildBlockUpdateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		blockType string
		text      string
	}{
		{
			name:      "paragraph",
			blockType: string(notionapi.BlockTypeParagraph),
			text:      "New paragraph",
		},
		{
			name:      "heading1",
			blockType: string(notionapi.BlockTypeHeading1),
			text:      "New heading",
		},
		{
			name:      "code",
			blockType: string(notionapi.BlockTypeCode),
			text:      "new code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := buildBlockUpdateRequest(tt.blockType, tt.text)
			if req == nil {
				t.Fatal("expected non-nil request")
			}

			// Verify the request has the correct field populated
			switch tt.blockType {
			case string(notionapi.BlockTypeParagraph):
				if req.Paragraph == nil {
					t.Error("expected Paragraph to be set")
				}
			case string(notionapi.BlockTypeHeading1):
				if req.Heading1 == nil {
					t.Error("expected Heading1 to be set")
				}
			case string(notionapi.BlockTypeCode):
				if req.Code == nil {
					t.Error("expected Code to be set")
				}
			}
		})
	}
}

func TestEditPageIntegration_LoadEditSave(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	block := testhelpers.NewParagraphBlock("Original text")
	mockClient.WithBlock(block)

	ep := NewEditPage(NewEditPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		PageID:       "page-123",
		BlockID:      "block-456",
	})

	// Step 1: Init and load block
	cmd := ep.Init()
	msg := cmd()
	model, _ := ep.Update(msg)
	epPtr := model.(*EditPage)
	ep = *epPtr

	if ep.loading {
		t.Error("expected loading to be false after load")
	}
	if ep.originalText != "Original text" {
		t.Errorf("expected originalText 'Original text', got %s", ep.originalText)
	}

	// Step 2: Trigger save
	saveMsg := components.SaveDraftMsg{
		BlockID:   "block-456",
		BlockType: string(notionapi.BlockTypeParagraph),
		Content:   "Updated text",
	}
	model, cmd = ep.Update(saveMsg)
	epPtr = model.(*EditPage)
	ep = *epPtr

	if !ep.saving {
		t.Error("expected saving to be true")
	}

	// Step 3: Execute save and handle result
	msg = cmd()
	model, _ = ep.Update(msg)
	epPtr = model.(*EditPage)
	ep = *epPtr

	if ep.saving {
		t.Error("expected saving to be false after save")
	}
	if !ep.saved {
		t.Error("expected saved to be true")
	}

	// Verify API calls
	if mockClient.GetBlockCallCount() != 1 {
		t.Errorf("expected 1 GetBlock call, got %d", mockClient.GetBlockCallCount())
	}
	if mockClient.UpdateBlockCallCount() != 1 {
		t.Errorf("expected 1 UpdateBlock call, got %d", mockClient.UpdateBlockCallCount())
	}
}
