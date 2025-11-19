package notion

import (
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
)

func TestConvertParagraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    *notionapi.ParagraphBlock
		expected string
	}{
		{
			name: "simple paragraph",
			block: &notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeParagraph,
				},
				Paragraph: notionapi.Paragraph{
					RichText: []notionapi.RichText{
						{PlainText: "Hello, world!"},
					},
				},
			},
			expected: "Hello, world!",
		},
		{
			name: "empty paragraph",
			block: &notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeParagraph,
				},
				Paragraph: notionapi.Paragraph{
					RichText: []notionapi.RichText{},
				},
			},
			expected: "",
		},
		{
			name:     "nil paragraph",
			block:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertParagraph(tt.block)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertHeadings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    notionapi.Block
		expected string
	}{
		{
			name: "heading 1",
			block: &notionapi.Heading1Block{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeHeading1,
				},
				Heading1: notionapi.Heading{
					RichText: []notionapi.RichText{
						{PlainText: "Main Title"},
					},
				},
			},
			expected: "# Main Title",
		},
		{
			name: "heading 2",
			block: &notionapi.Heading2Block{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeHeading2,
				},
				Heading2: notionapi.Heading{
					RichText: []notionapi.RichText{
						{PlainText: "Section Title"},
					},
				},
			},
			expected: "## Section Title",
		},
		{
			name: "heading 3",
			block: &notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeHeading3,
				},
				Heading3: notionapi.Heading{
					RichText: []notionapi.RichText{
						{PlainText: "Subsection"},
					},
				},
			},
			expected: "### Subsection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := convertBlock(tt.block, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertLists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    notionapi.Block
		listCtx  *listState
		expected string
	}{
		{
			name: "bulleted list item",
			block: &notionapi.BulletedListItemBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeBulletedListItem,
				},
				BulletedListItem: notionapi.ListItem{
					RichText: []notionapi.RichText{
						{PlainText: "First item"},
					},
				},
			},
			listCtx:  nil,
			expected: "- First item",
		},
		{
			name: "numbered list item first",
			block: &notionapi.NumberedListItemBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeNumberedListItem,
				},
				NumberedListItem: notionapi.ListItem{
					RichText: []notionapi.RichText{
						{PlainText: "Step one"},
					},
				},
			},
			listCtx:  &listState{counter: 1},
			expected: "1. Step one",
		},
		{
			name: "numbered list item third",
			block: &notionapi.NumberedListItemBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeNumberedListItem,
				},
				NumberedListItem: notionapi.ListItem{
					RichText: []notionapi.RichText{
						{PlainText: "Step three"},
					},
				},
			},
			listCtx:  &listState{counter: 3},
			expected: "3. Step three",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := convertBlock(tt.block, tt.listCtx)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    *notionapi.CodeBlock
		expected string
	}{
		{
			name: "go code block",
			block: &notionapi.CodeBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeCode,
				},
				Code: notionapi.Code{
					Language: "go",
					RichText: []notionapi.RichText{
						{PlainText: "fmt.Println(\"Hello\")"},
					},
				},
			},
			expected: "```go\nfmt.Println(\"Hello\")\n```",
		},
		{
			name: "python code block",
			block: &notionapi.CodeBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeCode,
				},
				Code: notionapi.Code{
					Language: "python",
					RichText: []notionapi.RichText{
						{PlainText: "print('Hello')"},
					},
				},
			},
			expected: "```python\nprint('Hello')\n```",
		},
		{
			name: "plain text code block",
			block: &notionapi.CodeBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeCode,
				},
				Code: notionapi.Code{
					Language: "plain text",
					RichText: []notionapi.RichText{
						{PlainText: "some text"},
					},
				},
			},
			expected: "```plain text\nsome text\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertCode(tt.block)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    *notionapi.QuoteBlock
		expected string
	}{
		{
			name: "single line quote",
			block: &notionapi.QuoteBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeQuote,
				},
				Quote: notionapi.Quote{
					RichText: []notionapi.RichText{
						{PlainText: "This is a quote"},
					},
				},
			},
			expected: "> This is a quote",
		},
		{
			name: "multi-line quote",
			block: &notionapi.QuoteBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeQuote,
				},
				Quote: notionapi.Quote{
					RichText: []notionapi.RichText{
						{PlainText: "First line\nSecond line"},
					},
				},
			},
			expected: "> First line\n> Second line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertQuote(tt.block)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertTodo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    *notionapi.ToDoBlock
		expected string
	}{
		{
			name: "checked todo",
			block: &notionapi.ToDoBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeToDo,
				},
				ToDo: notionapi.ToDo{
					RichText: []notionapi.RichText{
						{PlainText: "Complete task"},
					},
					Checked: true,
				},
			},
			expected: "- [x] Complete task",
		},
		{
			name: "unchecked todo",
			block: &notionapi.ToDoBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeToDo,
				},
				ToDo: notionapi.ToDo{
					RichText: []notionapi.RichText{
						{PlainText: "Pending task"},
					},
					Checked: false,
				},
			},
			expected: "- [ ] Pending task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertToDo(tt.block)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertRichText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     []notionapi.RichText
		expected string
	}{
		{
			name: "bold text",
			text: []notionapi.RichText{
				{
					PlainText: "bold",
					Annotations: &notionapi.Annotations{
						Bold: true,
					},
				},
			},
			expected: "**bold**",
		},
		{
			name: "italic text",
			text: []notionapi.RichText{
				{
					PlainText: "italic",
					Annotations: &notionapi.Annotations{
						Italic: true,
					},
				},
			},
			expected: "*italic*",
		},
		{
			name: "strikethrough text",
			text: []notionapi.RichText{
				{
					PlainText: "deleted",
					Annotations: &notionapi.Annotations{
						Strikethrough: true,
					},
				},
			},
			expected: "~~deleted~~",
		},
		{
			name: "code text",
			text: []notionapi.RichText{
				{
					PlainText: "code",
					Annotations: &notionapi.Annotations{
						Code: true,
					},
				},
			},
			expected: "`code`",
		},
		{
			name: "link text",
			text: []notionapi.RichText{
				{
					PlainText: "click here",
					Href:      "https://example.com",
				},
			},
			expected: "[click here](https://example.com)",
		},
		{
			name: "bold italic text",
			text: []notionapi.RichText{
				{
					PlainText: "emphasis",
					Annotations: &notionapi.Annotations{
						Bold:   true,
						Italic: true,
					},
				},
			},
			expected: "***emphasis***",
		},
		{
			name: "combined formatting",
			text: []notionapi.RichText{
				{PlainText: "Normal "},
				{
					PlainText: "bold",
					Annotations: &notionapi.Annotations{
						Bold: true,
					},
				},
				{PlainText: " and "},
				{
					PlainText: "italic",
					Annotations: &notionapi.Annotations{
						Italic: true,
					},
				},
			},
			expected: "Normal **bold** and *italic*",
		},
		{
			name:     "empty text",
			text:     []notionapi.RichText{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertRichText(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertDivider(t *testing.T) {
	t.Parallel()

	block := &notionapi.DividerBlock{
		BasicBlock: notionapi.BasicBlock{
			Type: notionapi.BlockTypeDivider,
		},
	}

	result, err := convertBlock(block, nil)
	assert.NoError(t, err)
	assert.Equal(t, "---", result)
}

func TestConvertImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    *notionapi.ImageBlock
		expected string
	}{
		{
			name: "external image with caption",
			block: &notionapi.ImageBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeImage,
				},
				Image: notionapi.Image{
					Type: notionapi.FileTypeExternal,
					External: &notionapi.FileObject{
						URL: "https://example.com/image.png",
					},
					Caption: []notionapi.RichText{
						{PlainText: "My Image"},
					},
				},
			},
			expected: "![My Image](https://example.com/image.png)",
		},
		{
			name: "external image without caption",
			block: &notionapi.ImageBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeImage,
				},
				Image: notionapi.Image{
					Type: notionapi.FileTypeExternal,
					External: &notionapi.FileObject{
						URL: "https://example.com/photo.jpg",
					},
					Caption: []notionapi.RichText{},
				},
			},
			expected: "![](https://example.com/photo.jpg)",
		},
		{
			name: "file image",
			block: &notionapi.ImageBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeImage,
				},
				Image: notionapi.Image{
					Type: notionapi.FileTypeFile,
					File: &notionapi.FileObject{
						URL: "https://s3.amazonaws.com/notion/file.png",
					},
					Caption: []notionapi.RichText{
						{PlainText: "Uploaded file"},
					},
				},
			},
			expected: "![Uploaded file](https://s3.amazonaws.com/notion/file.png)",
		},
		{
			name:     "nil image block",
			block:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertImage(tt.block)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertNesting(t *testing.T) {
	t.Parallel()

	// Test blocks with HasChildren flag
	block := &notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{
			Type:        notionapi.BlockTypeParagraph,
			HasChildren: true,
		},
		Paragraph: notionapi.Paragraph{
			RichText: []notionapi.RichText{
				{PlainText: "Parent block"},
			},
		},
	}

	assert.True(t, HasChildren(block))
	result, err := convertBlock(block, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Parent block", result)
}

func TestConvertEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		blocks   []notionapi.Block
		expected string
	}{
		{
			name:     "empty block slice",
			blocks:   []notionapi.Block{},
			expected: "",
		},
		{
			name:     "nil block slice",
			blocks:   nil,
			expected: "",
		},
		{
			name: "slice with nil blocks",
			blocks: []notionapi.Block{
				nil,
				nil,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertBlocksToMarkdown(tt.blocks)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertUnsupported(t *testing.T) {
	t.Parallel()

	// Test with an unsupported block type
	// Since we return empty string for unsupported types, it should not error
	block := &notionapi.TableBlock{
		BasicBlock: notionapi.BasicBlock{
			Type: "table",
		},
	}

	result, err := convertBlock(block, nil)
	assert.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestConvertMultiple(t *testing.T) {
	t.Parallel()

	blocks := []notionapi.Block{
		&notionapi.Heading1Block{
			BasicBlock: notionapi.BasicBlock{
				Type: notionapi.BlockTypeHeading1,
			},
			Heading1: notionapi.Heading{
				RichText: []notionapi.RichText{
					{PlainText: "Title"},
				},
			},
		},
		&notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Type: notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{
				RichText: []notionapi.RichText{
					{PlainText: "First paragraph."},
				},
			},
		},
		&notionapi.BulletedListItemBlock{
			BasicBlock: notionapi.BasicBlock{
				Type: notionapi.BlockTypeBulletedListItem,
			},
			BulletedListItem: notionapi.ListItem{
				RichText: []notionapi.RichText{
					{PlainText: "Item 1"},
				},
			},
		},
		&notionapi.BulletedListItemBlock{
			BasicBlock: notionapi.BasicBlock{
				Type: notionapi.BlockTypeBulletedListItem,
			},
			BulletedListItem: notionapi.ListItem{
				RichText: []notionapi.RichText{
					{PlainText: "Item 2"},
				},
			},
		},
		&notionapi.DividerBlock{
			BasicBlock: notionapi.BasicBlock{
				Type: notionapi.BlockTypeDivider,
			},
		},
		&notionapi.QuoteBlock{
			BasicBlock: notionapi.BasicBlock{
				Type: notionapi.BlockTypeQuote,
			},
			Quote: notionapi.Quote{
				RichText: []notionapi.RichText{
					{PlainText: "A wise quote"},
				},
			},
		},
	}

	expected := `# Title

First paragraph.

- Item 1
- Item 2
---

> A wise quote`

	result, err := ConvertBlocksToMarkdown(blocks)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestConvertBookmark(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		block    *notionapi.BookmarkBlock
		expected string
	}{
		{
			name: "bookmark with caption",
			block: &notionapi.BookmarkBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeBookmark,
				},
				Bookmark: notionapi.Bookmark{
					URL: "https://example.com",
					Caption: []notionapi.RichText{
						{PlainText: "Example Site"},
					},
				},
			},
			expected: "[Example Site](https://example.com)",
		},
		{
			name: "bookmark without caption",
			block: &notionapi.BookmarkBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeBookmark,
				},
				Bookmark: notionapi.Bookmark{
					URL:     "https://example.com/page",
					Caption: []notionapi.RichText{},
				},
			},
			expected: "[https://example.com/page](https://example.com/page)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertBookmark(tt.block)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertCallout(t *testing.T) {
	t.Parallel()

	warningEmoji := notionapi.Emoji("warning")

	tests := []struct {
		name     string
		block    *notionapi.CalloutBlock
		expected string
	}{
		{
			name: "callout with emoji",
			block: &notionapi.CalloutBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeCallout,
				},
				Callout: notionapi.Callout{
					Icon: &notionapi.Icon{
						Emoji: &warningEmoji,
					},
					RichText: []notionapi.RichText{
						{PlainText: "Important notice"},
					},
				},
			},
			expected: "> warning Important notice",
		},
		{
			name: "callout without emoji",
			block: &notionapi.CalloutBlock{
				BasicBlock: notionapi.BasicBlock{
					Type: notionapi.BlockTypeCallout,
				},
				Callout: notionapi.Callout{
					RichText: []notionapi.RichText{
						{PlainText: "Plain callout"},
					},
				},
			},
			expected: "> Plain callout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertCallout(tt.block)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("GetRichTextString", func(t *testing.T) {
		t.Parallel()

		text := []notionapi.RichText{
			{PlainText: "Hello "},
			{PlainText: "World"},
		}
		result := GetRichTextString(text)
		assert.Equal(t, "Hello World", result)
	})

	t.Run("GetRichTextString empty", func(t *testing.T) {
		t.Parallel()

		result := GetRichTextString([]notionapi.RichText{})
		assert.Equal(t, "", result)
	})

	t.Run("HasChildren true", func(t *testing.T) {
		t.Parallel()

		block := &notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				HasChildren: true,
			},
		}
		assert.True(t, HasChildren(block))
	})

	t.Run("HasChildren false", func(t *testing.T) {
		t.Parallel()

		block := &notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				HasChildren: false,
			},
		}
		assert.False(t, HasChildren(block))
	})

	t.Run("HasChildren nil", func(t *testing.T) {
		t.Parallel()

		assert.False(t, HasChildren(nil))
	})

	t.Run("GetBlockType", func(t *testing.T) {
		t.Parallel()

		block := &notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Type: notionapi.BlockTypeParagraph,
			},
		}
		assert.Equal(t, "paragraph", GetBlockType(block))
	})

	t.Run("GetBlockType nil", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "", GetBlockType(nil))
	})

	t.Run("ExtractLinks", func(t *testing.T) {
		t.Parallel()

		text := []notionapi.RichText{
			{PlainText: "text", Href: "https://example.com"},
			{PlainText: "no link"},
			{PlainText: "another", Href: "https://other.com"},
		}
		links := ExtractLinks(text)
		assert.Equal(t, []string{"https://example.com", "https://other.com"}, links)
	})

	t.Run("ExtractLinks empty", func(t *testing.T) {
		t.Parallel()

		links := ExtractLinks([]notionapi.RichText{})
		assert.Nil(t, links)
	})

	t.Run("FormatCode", func(t *testing.T) {
		t.Parallel()

		result := FormatCode("print('hello')", "python")
		assert.Equal(t, "```python\nprint('hello')\n```", result)
	})

	t.Run("FormatCode empty language", func(t *testing.T) {
		t.Parallel()

		result := FormatCode("some code", "")
		assert.Equal(t, "```text\nsome code\n```", result)
	})

	t.Run("CreateCheckbox checked", func(t *testing.T) {
		t.Parallel()

		result := CreateCheckbox(true, "Done")
		assert.Equal(t, "- [x] Done", result)
	})

	t.Run("CreateCheckbox unchecked", func(t *testing.T) {
		t.Parallel()

		result := CreateCheckbox(false, "To do")
		assert.Equal(t, "- [ ] To do", result)
	})
}

func TestConvertTableOfContents(t *testing.T) {
	t.Parallel()

	block := &notionapi.TableOfContentsBlock{
		BasicBlock: notionapi.BasicBlock{
			Type: notionapi.BlockTypeTableOfContents,
		},
	}

	result, err := convertBlock(block, nil)
	assert.NoError(t, err)
	assert.Equal(t, "[Table of Contents]", result)
}
