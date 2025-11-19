package testhelpers

import (
	"time"

	"github.com/jomei/notionapi"
)

// NewTestPage creates a test page with the given ID and title.
// Returns a valid notionapi.Page structure for hermetic testing.
func NewTestPage(id, title string) *notionapi.Page {
	now := time.Now()

	return &notionapi.Page{
		Object:         notionapi.ObjectTypePage,
		ID:             notionapi.ObjectID(id),
		CreatedTime:    now,
		LastEditedTime: now,
		Archived:       false,
		Properties: notionapi.Properties{
			"title": &notionapi.TitleProperty{
				Title: NewTestRichText(title),
			},
		},
		Parent: notionapi.Parent{
			Type: notionapi.ParentTypeDatabaseID,
		},
		URL: "https://www.notion.so/" + id,
	}
}

// NewTestPageWithBlocks creates a test page with blocks attached.
// The blocks are not actually stored in the page but returned separately.
// This is useful for creating complete test scenarios.
func NewTestPageWithBlocks(id, title string, blocks []notionapi.Block) (*notionapi.Page, []notionapi.Block) {
	page := NewTestPage(id, title)
	return page, blocks
}

// NewTestRichText creates a slice of RichText from a plain text content string.
func NewTestRichText(content string) []notionapi.RichText {
	return []notionapi.RichText{
		{
			Type:      notionapi.ObjectTypeText,
			PlainText: content,
			Text: &notionapi.Text{
				Content: content,
			},
			Annotations: &notionapi.Annotations{
				Bold:          false,
				Italic:        false,
				Strikethrough: false,
				Underline:     false,
				Code:          false,
				Color:         notionapi.ColorDefault,
			},
		},
	}
}

// NewTestRichTextWithAnnotations creates a RichText with specific formatting.
type RichTextOptions struct {
	Bold          bool
	Italic        bool
	Strikethrough bool
	Underline     bool
	Code          bool
	Color         notionapi.Color
	Href          string
}

// NewTestRichTextAdvanced creates a RichText with custom annotations.
func NewTestRichTextAdvanced(content string, opts RichTextOptions) []notionapi.RichText {
	color := opts.Color
	if color == "" {
		color = notionapi.ColorDefault
	}

	return []notionapi.RichText{
		{
			Type:      notionapi.ObjectTypeText,
			PlainText: content,
			Href:      opts.Href,
			Text: &notionapi.Text{
				Content: content,
				Link:    nil,
			},
			Annotations: &notionapi.Annotations{
				Bold:          opts.Bold,
				Italic:        opts.Italic,
				Strikethrough: opts.Strikethrough,
				Underline:     opts.Underline,
				Code:          opts.Code,
				Color:         color,
			},
		},
	}
}

// NewTestDatabase creates a database query response with sample pages.
func NewTestDatabase(dbID string) *notionapi.DatabaseQueryResponse {
	return &notionapi.DatabaseQueryResponse{
		Object: notionapi.ObjectTypeList,
		Results: []notionapi.Page{
			*NewTestPage("page-1", "First Page"),
			*NewTestPage("page-2", "Second Page"),
			*NewTestPage("page-3", "Third Page"),
			*NewTestPage("page-4", "Fourth Page"),
			*NewTestPage("page-5", "Fifth Page"),
		},
		HasMore:    false,
		NextCursor: "",
	}
}

// NewTestDatabaseEmpty creates an empty database query response.
func NewTestDatabaseEmpty(dbID string) *notionapi.DatabaseQueryResponse {
	return &notionapi.DatabaseQueryResponse{
		Object:     notionapi.ObjectTypeList,
		Results:    []notionapi.Page{},
		HasMore:    false,
		NextCursor: "",
	}
}

// NewTestDatabaseWithCursor creates a database response with pagination cursor.
func NewTestDatabaseWithCursor(dbID string, cursor string) *notionapi.DatabaseQueryResponse {
	return &notionapi.DatabaseQueryResponse{
		Object: notionapi.ObjectTypeList,
		Results: []notionapi.Page{
			*NewTestPage("page-1", "Page One"),
			*NewTestPage("page-2", "Page Two"),
		},
		HasMore:    true,
		NextCursor: notionapi.Cursor(cursor),
	}
}

// NewTestBlockList creates a list of test blocks with the specified count.
// Generates a variety of block types for comprehensive testing.
func NewTestBlockList(count int) []notionapi.Block {
	blocks := make([]notionapi.Block, 0, count)
	blockTypes := []func() notionapi.Block{
		func() notionapi.Block { return NewParagraphBlock("Sample paragraph text") },
		func() notionapi.Block { return NewHeading1Block("Main Heading") },
		func() notionapi.Block { return NewHeading2Block("Sub Heading") },
		func() notionapi.Block { return NewBulletedListItemBlock("List item") },
		func() notionapi.Block { return NewCodeBlock("go", "fmt.Println(\"Hello\")") },
	}

	for i := 0; i < count; i++ {
		block := blockTypes[i%len(blockTypes)]()
		blocks = append(blocks, block)
	}

	return blocks
}

// NewParagraphBlock creates a paragraph block with the given text.
func NewParagraphBlock(text string) *notionapi.ParagraphBlock {
	return &notionapi.ParagraphBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeParagraph),
		Paragraph: notionapi.Paragraph{
			RichText: NewTestRichText(text),
			Color:    string(notionapi.ColorDefault),
		},
	}
}

// NewHeading1Block creates a heading 1 block with the given text.
func NewHeading1Block(text string) *notionapi.Heading1Block {
	return &notionapi.Heading1Block{
		BasicBlock: newBasicBlock(notionapi.BlockTypeHeading1),
		Heading1: notionapi.Heading{
			RichText:     NewTestRichText(text),
			Color:        string(notionapi.ColorDefault),
			IsToggleable: false,
		},
	}
}

// NewHeading2Block creates a heading 2 block with the given text.
func NewHeading2Block(text string) *notionapi.Heading2Block {
	return &notionapi.Heading2Block{
		BasicBlock: newBasicBlock(notionapi.BlockTypeHeading2),
		Heading2: notionapi.Heading{
			RichText:     NewTestRichText(text),
			Color:        string(notionapi.ColorDefault),
			IsToggleable: false,
		},
	}
}

// NewHeading3Block creates a heading 3 block with the given text.
func NewHeading3Block(text string) *notionapi.Heading3Block {
	return &notionapi.Heading3Block{
		BasicBlock: newBasicBlock(notionapi.BlockTypeHeading3),
		Heading3: notionapi.Heading{
			RichText:     NewTestRichText(text),
			Color:        string(notionapi.ColorDefault),
			IsToggleable: false,
		},
	}
}

// NewBulletedListItemBlock creates a bulleted list item block.
func NewBulletedListItemBlock(text string) *notionapi.BulletedListItemBlock {
	return &notionapi.BulletedListItemBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeBulletedListItem),
		BulletedListItem: notionapi.ListItem{
			RichText: NewTestRichText(text),
			Color:    string(notionapi.ColorDefault),
		},
	}
}

// NewNumberedListItemBlock creates a numbered list item block.
func NewNumberedListItemBlock(text string) *notionapi.NumberedListItemBlock {
	return &notionapi.NumberedListItemBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeNumberedListItem),
		NumberedListItem: notionapi.ListItem{
			RichText: NewTestRichText(text),
			Color:    string(notionapi.ColorDefault),
		},
	}
}

// NewToDoBlock creates a to-do block with the given text and checked state.
func NewToDoBlock(text string, checked bool) *notionapi.ToDoBlock {
	return &notionapi.ToDoBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeToDo),
		ToDo: notionapi.ToDo{
			RichText: NewTestRichText(text),
			Checked:  checked,
			Color:    string(notionapi.ColorDefault),
		},
	}
}

// NewCodeBlock creates a code block with the given language and code content.
func NewCodeBlock(language, code string) *notionapi.CodeBlock {
	return &notionapi.CodeBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeCode),
		Code: notionapi.Code{
			RichText: NewTestRichText(code),
			Language: language,
		},
	}
}

// NewQuoteBlock creates a quote block with the given text.
func NewQuoteBlock(text string) *notionapi.QuoteBlock {
	return &notionapi.QuoteBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeQuote),
		Quote: notionapi.Quote{
			RichText: NewTestRichText(text),
			Color:    string(notionapi.ColorDefault),
		},
	}
}

// NewDividerBlock creates a divider block.
func NewDividerBlock() *notionapi.DividerBlock {
	return &notionapi.DividerBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeDivider),
		Divider:    struct{}{},
	}
}

// NewCalloutBlock creates a callout block with the given emoji and text.
func NewCalloutBlock(emoji, text string) *notionapi.CalloutBlock {
	emojiValue := notionapi.Emoji(emoji)
	return &notionapi.CalloutBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeCallout),
		Callout: notionapi.Callout{
			RichText: NewTestRichText(text),
			Icon: &notionapi.Icon{
				Type:  "emoji",
				Emoji: &emojiValue,
			},
			Color: string(notionapi.ColorGrayBackground),
		},
	}
}

// NewToggleBlock creates a toggle block with the given text.
func NewToggleBlock(text string) *notionapi.ToggleBlock {
	return &notionapi.ToggleBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeToggle),
		Toggle: notionapi.Toggle{
			RichText: NewTestRichText(text),
			Color:    string(notionapi.ColorDefault),
		},
	}
}

// NewImageBlock creates an image block with the given URL.
func NewImageBlock(url string, caption string) *notionapi.ImageBlock {
	return &notionapi.ImageBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeImage),
		Image: notionapi.Image{
			Type: notionapi.FileTypeExternal,
			External: &notionapi.FileObject{
				URL: url,
			},
			Caption: NewTestRichText(caption),
		},
	}
}

// NewBookmarkBlock creates a bookmark block with the given URL.
func NewBookmarkBlock(url string, caption string) *notionapi.BookmarkBlock {
	return &notionapi.BookmarkBlock{
		BasicBlock: newBasicBlock(notionapi.BlockTypeBookmark),
		Bookmark: notionapi.Bookmark{
			URL:     url,
			Caption: NewTestRichText(caption),
		},
	}
}

// NewGetChildrenResponse creates a GetChildrenResponse with the given blocks.
func NewGetChildrenResponse(blocks []notionapi.Block) *notionapi.GetChildrenResponse {
	return &notionapi.GetChildrenResponse{
		Object:     notionapi.ObjectTypeList,
		Results:    blocks,
		HasMore:    false,
		NextCursor: "",
	}
}

// NewGetChildrenResponseWithCursor creates a paginated GetChildrenResponse.
func NewGetChildrenResponseWithCursor(blocks []notionapi.Block, cursor string) *notionapi.GetChildrenResponse {
	return &notionapi.GetChildrenResponse{
		Object:     notionapi.ObjectTypeList,
		Results:    blocks,
		HasMore:    true,
		NextCursor: cursor,
	}
}

// newBasicBlock creates a BasicBlock with common fields populated.
func newBasicBlock(blockType notionapi.BlockType) notionapi.BasicBlock {
	now := time.Now()
	return notionapi.BasicBlock{
		Object:         notionapi.ObjectTypeBlock,
		ID:             notionapi.BlockID(generateTestID()),
		Type:           blockType,
		CreatedTime:    &now,
		LastEditedTime: &now,
		HasChildren:    false,
		Archived:       false,
	}
}

// testIDCounter is used to generate unique IDs for test blocks.
var testIDCounter int

// generateTestID generates a unique test ID.
func generateTestID() string {
	testIDCounter++
	return "test-block-" + itoa(testIDCounter)
}

// itoa converts an integer to a string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var buf [20]byte
	pos := len(buf)
	negative := i < 0
	if negative {
		i = -i
	}

	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}

	if negative {
		pos--
		buf[pos] = '-'
	}

	return string(buf[pos:])
}

// ResetTestIDCounter resets the test ID counter. Useful for deterministic tests.
func ResetTestIDCounter() {
	testIDCounter = 0
}
